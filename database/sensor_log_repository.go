package database

import (
	"time"

	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"
)

// ==========================================
// 🌟 ฟังก์ชัน CRUD สำหรับบันทึกข้อมูลเซนเซอร์ (Sensor Log)
// ==========================================

// CreateSensorLog บันทึกค่าเซนเซอร์ใหม่ และอัปเดตสถานะอุปกรณ์ (Device) ในเวลาเดียวกัน
func CreateSensorLog(log *models.SensorLog) error {
	// ใช้ Transaction ป้องกันข้อมูลพัง ถ้าอัปเดตตัวใดตัวหนึ่งไม่ผ่าน ระบบจะยกเลิกทั้งหมด
	return DB.Transaction(func(tx *gorm.DB) error {
		
		// 1. บันทึกประวัติเซนเซอร์ลงตาราง sensor_log
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		// 2. ไปอัปเดตสถานะและเวลาอัปเดตล่าสุดที่ตาราง device ตัวแม่
		if err := tx.Model(&models.Device{}).
			Where("device_id = ?", log.DeviceID).
			Updates(map[string]interface{}{
				"current_status": "Online",
				"last_update":    time.Now(),
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetSensorLogsByDeviceID ดึงประวัติค่าเซนเซอร์ย้อนหลังของอุปกรณ์นั้นๆ (เผื่อไว้ทำกราฟในแอป)
func GetSensorLogsByDeviceID(deviceID int, limit int) ([]models.SensorLog, error) {
	var logs []models.SensorLog
	// ดึงข้อมูลโดยเรียงจากเวลาล่าสุดลงไป และจำกัดจำนวนแถว (limit)
	err := DB.Where("device_id = ?", deviceID).
		Order("timestamp desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func GetDevicesByCoop(coopID int) ([]models.Device, error) {
	var devices []models.Device
	
	// ใช้คำสั่ง GORM ดึงข้อมูลอุปกรณ์ที่มี coop_id ตรงกับที่ส่งมา
	err := DB.Where("coop_id = ?", coopID).Find(&devices).Error
	
	return devices, err
}

