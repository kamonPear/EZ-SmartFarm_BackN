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

// MotionAlertRow คือ 1 เหตุการณ์ที่เซนเซอร์ตรวจจับการเคลื่อนไหว (PIR) ที่วงกบประตูคอก
// ตรวจจับได้ - มาจากแถว sensor_log ปกติ แค่กรองเฉพาะอุปกรณ์ที่ชื่อ/ประเภทมีคำว่า
// "pir" อยู่ (ตามแบบแผนเดียวกับที่ฝั่งแอปใช้เลือกไอคอนอุปกรณ์อยู่แล้ว - ดู
// Main_DeviceSummary.dart ที่เช็ค lower.contains('pir'))
type MotionAlertRow struct {
	CoopID     int       `json:"coop_id"`
	CoopName   string    `json:"coop_name"`
	DeviceID   int32     `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
}

// GetMotionAlertsForUser คืนรายการเหตุการณ์ตรวจจับความเคลื่อนไหว (จากเซนเซอร์ PIR
// ที่วงกบประตู) ของทุกคอกที่ userID เป็นเจ้าของ นับตั้งแต่เวลา since เป็นต้นมา
// เรียงจากล่าสุดไปเก่าสุด - ฝั่งหน้าบ้านเอาไปจัดกลุ่มแสดงผลเป็น "คอกที่ X มีอะไร
// เดินผ่าน" ต่อเอง
func GetMotionAlertsForUser(userID int, since time.Time) ([]MotionAlertRow, error) {
	coops, err := GetAllCoops(userID)
	if err != nil {
		return nil, err
	}
	if len(coops) == 0 {
		return []MotionAlertRow{}, nil
	}

	coopIDs := make([]int, len(coops))
	coopNameByID := make(map[int]string, len(coops))
	for i, c := range coops {
		coopIDs[i] = c.CoopID
		coopNameByID[c.CoopID] = c.NameCoop
	}

	var devices []models.Device
	if err := DB.Where(
		"coop_id IN ? AND (LOWER(name) LIKE ? OR LOWER(device_type) LIKE ?)",
		coopIDs, "%pir%", "%pir%",
	).Find(&devices).Error; err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return []MotionAlertRow{}, nil
	}

	deviceIDs := make([]int32, len(devices))
	deviceNameByID := make(map[int32]string, len(devices))
	coopIDByDeviceID := make(map[int32]int, len(devices))
	for i, d := range devices {
		deviceIDs[i] = d.DeviceID
		deviceNameByID[d.DeviceID] = d.Name
		coopIDByDeviceID[d.DeviceID] = int(d.CoopID)
	}

	var logs []models.SensorLog
	if err := DB.Where("device_id IN ? AND timestamp >= ?", deviceIDs, since).
		Order("timestamp desc").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	result := make([]MotionAlertRow, 0, len(logs))
	for _, l := range logs {
		cid := coopIDByDeviceID[l.DeviceID]
		result = append(result, MotionAlertRow{
			CoopID:     cid,
			CoopName:   coopNameByID[cid],
			DeviceID:   l.DeviceID,
			DeviceName: deviceNameByID[l.DeviceID],
			Value:      l.Value,
			Timestamp:  l.Timestamp,
		})
	}
	return result, nil
}

