package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"EZ-SmartFarm_BachN/database" // ใช้สำหรับ StartOfflineChecker
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// โครงสร้างข้อมูลที่รับมาจาก Arduino
type SensorPayload struct {
	CoopID     int     `json:"coop_id"`
	SlotIndex  int     `json:"slot_index"`  // ตำแหน่งช่องที่วางอุปกรณ์ไว้ (ใช้จับคู่อุปกรณ์แทนชื่อ เพราะชื่อซ้ำกันได้)
	DeviceName string  `json:"device_name"` // ใช้แค่ log/debug ไม่ใช้จับคู่แล้ว
	Value      float64 `json:"value"`
}

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
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		fmt.Println("⚠️ [MQTT] Connection lost:", err)
	})
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
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

func StartOfflineChecker() {
	for {
		time.Sleep(5 * time.Second)
		db := database.GetDB()
		if db != nil {
			db.Exec("UPDATE device SET current_status = 'Offline' WHERE last_update < NOW() - INTERVAL 15 SECOND")
		}
	}
}