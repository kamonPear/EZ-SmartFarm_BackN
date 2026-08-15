package services

import (
	"fmt"
	"time"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// ProcessAndSaveSensorLog รับข้อมูลจาก MQTT มาตรวจสอบและบันทึกลงฐานข้อมูล
func ProcessAndSaveSensorLog(payload SensorPayload) {
	db := database.GetDB()
	if db == nil {
		fmt.Println("❌ [DB Error] ไม่สามารถเชื่อมต่อฐานข้อมูลได้")
		return
	}

	// 1. ค้นหา Device จากฐานข้อมูลด้วย CoopID และ slot_index (ไม่ใช้ชื่อแล้ว เพราะชื่อซ้ำกันได้ในคนละ slot)
	var device models.Device
	result := db.Where("coop_id = ? AND slot_index = ?", payload.CoopID, payload.SlotIndex).First(&device)

	if result.Error != nil {
		fmt.Printf("⚠️ [DB Warning] หาอุปกรณ์ที่ช่อง %d ในคอก %d ไม่เจอ (ชื่อที่ส่งมา: '%s')\n", payload.SlotIndex, payload.CoopID, payload.DeviceName)
		return
	}

	// 2. อัปเดตตาราง devices ให้สถานะเป็น Online และบันทึกเวลาล่าสุด
	db.Model(&device).Updates(map[string]interface{}{
		"current_status": "Online",
		"last_update":    time.Now(),
	})

	// 3. จัดการตาราง sensor_log (อัปเดตช่องเดิม หรือ สร้างใหม่ถ้าไม่เคยมี)
	var logData models.SensorLog
	err := db.Where("device_id = ? AND coop_id = ?", device.DeviceID, payload.CoopID).First(&logData).Error

	if err != nil {
		// กรณีที่ยังไม่เคยมีข้อมูลมาก่อน -> ให้สร้างแถวใหม่
		logData = models.SensorLog{
			CoopID:    int32(payload.CoopID),
			DeviceID:  device.DeviceID,
			Value:     payload.Value,
			Timestamp: time.Now(),
		}
		if err := db.Create(&logData).Error; err != nil {
			fmt.Println("❌ [DB Error] สร้างข้อมูลใหม่ไม่ได้:", err)
		} else {
			fmt.Printf("✅ [DB Success] สร้างแถวใหม่ให้ '%s' | ค่า: %.2f\n", payload.DeviceName, payload.Value)
		}
	} else {
		// กรณีที่มีข้อมูลอยู่แล้ว -> ให้โยนค่าใหม่อัปเดตทับช่องเดิม
		if err := db.Model(&logData).Updates(map[string]interface{}{
			"value":     payload.Value,
			"timestamp": time.Now(),
		}).Error; err != nil {
			fmt.Println("❌ [DB Error] อัปเดตข้อมูลเดิมไม่ได้:", err)
		} else {
			fmt.Printf("✅ [DB Success] อัปเดต '%s' ทับช่องเดิม | ค่า: %.2f\n", payload.DeviceName, payload.Value)
		}
	}
}