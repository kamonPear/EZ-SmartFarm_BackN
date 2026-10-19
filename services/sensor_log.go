// =====================================================================================
// ไฟล์ services/sensor_log.go - บันทึกค่าที่บอร์ดส่งมาทาง MQTT ลงฐานข้อมูล
// =====================================================================================
//
// เส้นทางข้อมูล: บอร์ด ESP32 -> MQTT (farm/sensors/data) -> StartMQTTWorker -> ProcessAndSaveSensorLog -> MySQL
// แล้วหน้าเว็บ/แอปดึงค่าล่าสุดผ่าน API (handlers/sensor_log.go) ไปแสดงผล
package services

import (
	"fmt"
	"time"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// ProcessAndSaveSensorLog รับค่า 1 ข้อความจากบอร์ด แล้วอัปเดตฐานข้อมูล
//
// ขั้นตอน:
//  1. หาอุปกรณ์จาก (coop_id, slot_index) ถ้าไม่เจอ (เช่นผู้ใช้ลบอุปกรณ์ไปแล้ว) log เตือนแล้วข้าม
//  2. อัปเดตสถานะอุปกรณ์และเวลาล่าสุด (last_update)
//     - เซนเซอร์ : ส่งข้อมูลมา = Online
//     - พัดลม/หลอดไฟ (Actuator) : ค่า > 0 = กำลังทำงาน (Online), ค่า <= 0 = ปิดอยู่ (Offline)
//     last_update นี้คือสิ่งที่ StartOfflineChecker ใช้ตัดสินว่าอุปกรณ์เงียบไปนานหรือยัง
//  3. ตาราง sensor_log เก็บ "ค่าล่าสุด" 1 แถวต่ออุปกรณ์ (ไม่ใช่เก็บทุกค่า)
//     - ยังไม่มีแถว -> สร้างใหม่
//     - มีแล้ว -> เขียนค่าใหม่ทับ
//     ทำไมเขียนทับ: บอร์ดส่งข้อมูลทุกไม่กี่วินาที ถ้าเก็บทุกค่าตารางจะโตเร็วมาก
//     และหน้าจอใช้แค่ค่าปัจจุบัน
// ถูกเรียกจาก: services/mqtt_service.go (callback ตอนมีข้อความเข้า)
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

	// 2. อัปเดตตาราง devices แล้วบันทึกเวลาล่าสุด
	// เซนเซอร์: ส่งข้อมูลมา = Online
	// อุปกรณ์สั่งงาน (พัดลม/หลอดไฟ): Online เมื่อกำลังทำงาน (value > 0) และ Offline เมื่อหยุดทำงาน
	// (บอร์ดส่งสถานะมาทุกรอบ ทั้งตอนติดและตอนดับ)
	status := "Online"
	if device.DeviceType == "Actuator" && payload.Value <= 0 {
		status = "Offline"
	}
	if device.CurrentStatus != status {
		fmt.Printf("🔺 [Status] คอก %d ช่อง %d '%s': %s -> %s (เงียบมา %.1f วินาที, ค่า=%.2f)\n",
			payload.CoopID, payload.SlotIndex, device.Name, device.CurrentStatus, status,
			time.Since(device.LastUpdate).Seconds(), payload.Value)
	}
	db.Model(&device).Updates(map[string]interface{}{
		"current_status": status,
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
