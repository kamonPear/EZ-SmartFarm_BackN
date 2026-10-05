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

// โครงสร้างข้อมูลที่รับมาจาก Arduino
type SensorPayload struct {
	CoopID     int     `json:"coop_id"`
	SlotIndex  int     `json:"slot_index"`  // ตำแหน่งช่องที่วางอุปกรณ์ไว้ (ใช้จับคู่อุปกรณ์แทนชื่อ เพราะชื่อซ้ำกันได้)
	DeviceName string  `json:"device_name"` // ใช้แค่ log/debug ไม่ใช้จับคู่แล้ว
	Value      float64 `json:"value"`
}

// mqttClient คือ client ตัวเดียวกับที่ StartMQTTWorker สร้าง ใช้ publish กลับไปหาอุปกรณ์
var mqttClient mqtt.Client

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

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// offlineAfterSeconds คือเวลาที่ไม่ได้รับข้อมูลจากอุปกรณ์ก่อนจะถือว่า Offline
// ถ้ายังส่งข้อมูลมาเรื่อยๆ จะเป็น Online (ตั้งใน sensor_log.go ทุกครั้งที่รับข้อความ)
// แต่ถ้าเงียบเกิน 30 วินาทีจะเป็น Offline แล้วกลับเป็น Online ทันทีเมื่อมีข้อมูลเข้ามาอีก
const offlineAfterSeconds = 30

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