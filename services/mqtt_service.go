// =====================================================================================
// ไฟล์ services/mqtt_service.go - เชื่อมต่อ MQTT broker เพื่อรับค่าจากบอร์ด ESP32 และตรวจสถานะ Offline
// =====================================================================================
//
// MQTT คืออะไร: โปรโตคอลส่งข้อความแบบ publish/subscribe ที่เบามาก เหมาะกับอุปกรณ์ IoT
//  - บอร์ด ESP32 ในคอก "publish" ค่าเซนเซอร์ไปที่ topic farm/sensors/data
//  - backend ตัวนี้ "subscribe" topic นั้น เมื่อมีข้อความเข้ามาจะบันทึกลงฐานข้อมูล
//  - backend ยัง publish ค่าตั้งค่ากลับไปหาบอร์ดด้วย (เลขช่องอุปกรณ์, ค่ามาตรฐานอุณหภูมิ/แอมโมเนีย)
//  ตัวกลางที่รับส่งข้อความ (broker) แยกเป็นเซิร์ฟเวอร์ต่างหาก (ตั้งค่าผ่าน MQTT_BROKER_HOST ฯลฯ)
//
// ทำไมใช้ MQTT ไม่ให้บอร์ดยิง HTTP ตรง: เชื่อมต่อค้างไว้ได้ ส่งถี่ได้โดยไม่เปลืองแรง
// และสื่อสารได้สองทาง (backend ส่งค่ากลับไปหาบอร์ดได้)
// ถูกเรียกจาก: main.go (go services.StartMQTTWorker() และ go services.StartOfflineChecker())
package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"EZ-SmartFarm_BachN/database" // ใช้สำหรับ StartOfflineChecker
	"EZ-SmartFarm_BachN/models"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SensorPayload รูปแบบ JSON ที่บอร์ด ESP32 ส่งมาใน topic farm/sensors/data
// เช่น {"coop_id":1, "slot_index":4, "device_name":"DHT22", "value":31.5}
//  - CoopID     : คอกที่บอร์ดติดตั้งอยู่
//  - SlotIndex  : เลขอุปกรณ์ในคอก ใช้หาว่าค่านี้เป็นของอุปกรณ์ตัวไหน (ไม่ใช้ชื่อ เพราะชื่อซ้ำกันได้)
//  - DeviceName : ไว้ดูใน log เท่านั้น
//  - Value      : ค่าที่วัดได้ (เซนเซอร์) หรือสถานะ 1/0 (พัดลม/หลอดไฟ)
type SensorPayload struct {
	CoopID     int     `json:"coop_id"`
	SlotIndex  int     `json:"slot_index"`  // ตำแหน่งช่องที่วางอุปกรณ์ไว้ (ใช้จับคู่อุปกรณ์แทนชื่อ เพราะชื่อซ้ำกันได้)
	DeviceName string  `json:"device_name"` // ใช้แค่ log/debug ไม่ใช้จับคู่แล้ว
	Value      float64 `json:"value"`
}

// mqttClient ตัวเชื่อมต่อ MQTT ตัวเดียวของทั้งโปรแกรม สร้างใน StartMQTTWorker
// เก็บเป็นตัวแปรระดับ package เพื่อให้ slot_publisher.go และ threshold_publisher.go ใช้ publish กลับไปหาบอร์ดได้
var mqttClient mqtt.Client

// StartMQTTWorker ตั้งค่าและเชื่อมต่อ MQTT broker (รันเป็น goroutine จาก main.go)
//
// ขั้นตอน:
//  1. อ่านค่าการเชื่อมต่อจาก environment (host, port, user, password, ใช้ TLS ไหม)
//     ถ้าไม่ได้ตั้ง ใช้ค่าเริ่มต้น เหมาะกับการทดสอบในเครือข่ายบ้าน
//  2. ตั้ง client ID (ปรับได้ด้วย MQTT_CLIENT_ID) เพราะ broker ให้ ID เดียวกันต่อได้ทีละ 1 การเชื่อมต่อ
//  3. ตั้ง callback ไว้ 3 ตัว:
//     - เมื่อมีข้อความเข้า : แปลง JSON เป็น SensorPayload แล้วส่งให้ ProcessAndSaveSensorLog บันทึก
//     - เมื่อเชื่อมต่อสำเร็จ : subscribe farm/sensors/data แล้วส่งค่ามาตรฐานและเลขช่องให้ทุกคอกใหม่
//       (ทำใน goroutine เพราะห้ามรอผลภายใน callback ของไลบรารี paho)
//     - เมื่อหลุด : แค่ log เพราะเปิด AutoReconnect ไว้ ไลบรารีจะต่อใหม่ให้เอง
//  4. เชื่อมต่อ ถ้าล้มก็ log แล้วจบ (เซิร์ฟเวอร์ส่วนอื่นยังทำงานต่อได้)
func StartMQTTWorker() {
	brokerHost := getEnv("MQTT_BROKER_HOST", "192.168.0.102")
	brokerPort := getEnv("MQTT_BROKER_PORT", "1883")
	username := getEnv("MQTT_USERNAME", "")
	password := getEnv("MQTT_PASSWORD", "")
	useTLS := getEnv("MQTT_USE_TLS", "false") == "true"

	scheme := "tcp"
	if useTLS {
		scheme = "tls"
	}

	// ปรับ client ID ผ่าน MQTT_CLIENT_ID ได้เวลารันทดสอบบนเครื่อง (local) พร้อมกับที่ Render ยังรันอยู่
	// เพราะ broker อนุญาตแค่ 1 session ต่อ client ID เดียวกัน - ถ้าใช้ ID ซ้ำกัน อีกฝั่งจะหลุดการเชื่อมต่อทันที
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%s", scheme, brokerHost, brokerPort))
	opts.SetClientID(getEnv("MQTT_CLIENT_ID", "ez_farm_mqtt_worker"))

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	if useTLS {
		opts.SetTLSConfig(&tls.Config{})
	}

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("📥 [MQTT] Received on %s: %s\n", msg.Topic(), string(msg.Payload()))

		var payload SensorPayload
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			fmt.Println("❌ [MQTT Error] JSON ไม่ถูกต้อง:", err)
			return
		}

		// 🚀 โยน Payload ไปให้ฟังก์ชันในไฟล์ sensor_log.go เป็นคนจัดการฐานข้อมูล
		ProcessAndSaveSensorLog(payload)
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		fmt.Println("✅ [MQTT] Connected (or reconnected) to broker")
		if token := client.Subscribe("farm/sensors/data", 1, nil); token.Wait() && token.Error() != nil {
			fmt.Println("❌ [MQTT Subscribe Error]", token.Error())
		} else {
			fmt.Println("📡 Subscribed to 'farm/sensors/data'")
		}
		// ส่งค่ามาตรฐานล่าสุดให้ทุกคอกอีกรอบ (retained) กันกรณี broker ล้างค่าเก่าไป
		// ทำใน goroutine เพราะห้ามรอ token ภายใน callback ของ paho
		go PublishAllThresholds()
		go PublishAllSlots()
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		fmt.Println("⚠️ [MQTT] Connection lost:", err)
	})
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	mqttClient = client // เก็บไว้ให้ PublishThreshold ใช้ส่งค่ามาตรฐานกลับไปหา Arduino
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("❌ [MQTT Connection Error]", token.Error())
		return
	}
}

// getEnv อ่านค่า environment variable ถ้าไม่ได้ตั้งไว้ (ค่าว่าง) ให้ใช้ defaultValue แทน
// ถูกเรียกจาก: StartMQTTWorker
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// offlineAfterSeconds ถ้าอุปกรณ์เงียบ (ไม่ส่งข้อมูลมา) นานเกิน 30 วินาที จะถือว่า Offline
// บอร์ดส่งข้อมูลทุกไม่กี่วินาที 30 วินาทีจึงเผื่อเน็ตสะดุดได้โดยไม่ตีเป็น Offline เร็วเกินไป
// พอมีข้อมูลเข้ามาใหม่ จะกลับเป็น Online ทันที (ใน sensor_log.go)
const offlineAfterSeconds = 30

// StartOfflineChecker วนตรวจทุก 5 วินาที ว่ามีอุปกรณ์ไหนเงียบเกิน offlineAfterSeconds
// ถ้ามี ตั้งสถานะเป็น Offline (หน้าสถานะอุปกรณ์จะแสดงสีแดง/ออฟไลน์)
//
// ทำไมคำนวณเวลาตัด (cutoff) ฝั่ง Go แทน NOW() ของ MySQL:
// last_update ถูกเขียนด้วยเวลาของ Go ถ้าเครื่อง backend (เช่น Render ใช้ UTC) กับ MySQL (เวลาไทย)
// คนละ timezone เวลาจะเหลื่อมกันหลายชั่วโมง ทำให้อุปกรณ์ไม่เคยถูกตีเป็น Offline
// ก่อนอัปเดต จะ log รายชื่ออุปกรณ์ที่ถูกตีเป็น Offline ไว้ไล่หาสาเหตุ
// ถูกเรียกจาก: main.go (go services.StartOfflineChecker() รันตลอดอายุโปรแกรม)
func StartOfflineChecker() {
	for {
		time.Sleep(5 * time.Second)
		db := database.GetDB()
		if db != nil {
			// คำนวณเวลาตัดที่ฝั่ง Go แทน NOW() ของ MySQL เพราะ last_update ถูกเขียนด้วย time.Now()
			// ของ Go (loc=Local) ถ้า timezone ของเครื่อง backend กับของ MySQL ไม่ตรงกัน (เช่นรันบน
			// Render เป็น UTC แต่ DB เป็นเวลาไทย) เวลาจะเหลื่อมกันหลายชั่วโมง อุปกรณ์เลยไม่ถูกตีเป็น Offline
			cutoff := time.Now().Add(-offlineAfterSeconds * time.Second)
			// log ว่าตัวไหนถูกตีเป็น Offline เพราะเงียบมานานแค่ไหน ไว้ไล่สาเหตุสถานะติดๆ ดับๆ
			var stale []models.Device
			db.Where("last_update < ? AND current_status <> 'Offline'", cutoff).Find(&stale)
			for _, d := range stale {
				fmt.Printf("🔻 [Offline] คอก %d ช่อง %d '%s' เงียบมา %.1f วินาที (last_update=%s)\n",
					d.CoopID, d.SlotIndex, d.Name, time.Since(d.LastUpdate).Seconds(), d.LastUpdate.Format("15:04:05"))
			}
			if len(stale) > 0 {
				db.Exec("UPDATE device SET current_status = 'Offline' WHERE last_update < ? AND current_status <> 'Offline'", cutoff)
			}
		}
	}
}
