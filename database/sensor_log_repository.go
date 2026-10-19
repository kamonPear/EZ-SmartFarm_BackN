// =====================================================================================
// ไฟล์ database/sensor_log_repository.go - ค่าที่วัดได้จากเซนเซอร์ และการแจ้งเตือนความเคลื่อนไหว
// =====================================================================================
//
// บอร์ด ESP32 ในคอกส่งค่าที่วัดได้มาเป็นระยะ (อุณหภูมิ, แอมโมเนีย, PIR ตรวจจับการเคลื่อนไหว)
// ค่าเก็บในตาราง sensor_log และสถานะอุปกรณ์ (Online/Offline) อยู่ในตาราง device
// ถูกเรียกจาก: handlers/sensor_log.go และ services/ (MQTT)
package database

import (
	"time"

	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"
)

// ==========================================
// 🌟 ฟังก์ชัน CRUD สำหรับบันทึกข้อมูลเซนเซอร์ (Sensor Log)
// ==========================================

// CreateSensorLog บันทึกค่าเซนเซอร์ใหม่ และอัปเดตสถานะอุปกรณ์เป็น Online พร้อมกัน
//
// ใช้ Transaction (DB.Transaction): 2 คำสั่งด้านในต้องสำเร็จทั้งคู่ ถ้าตัวใดล้ม จะยกเลิกทั้งหมด (rollback)
// กันข้อมูลครึ่ง ๆ กลาง ๆ เช่น มีค่าเซนเซอร์แต่สถานะอุปกรณ์ไม่อัปเดต
//  1. insert ค่าลงตาราง sensor_log
//  2. อัปเดต device: current_status = "Online", last_update = ตอนนี้
//     (services/ มีตัวเช็กออฟไลน์ ที่จะเปลี่ยนเป็น Offline ถ้าไม่ได้ข้อมูลนานเกินกำหนด)
//
// ถูกเรียกจาก: handlers/sensor_log.go -> ReceiveSensorDataHandler (POST /api/sensor-logs)
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

// GetSensorLogsByDeviceID ดึงค่าย้อนหลังของอุปกรณ์ตัวหนึ่ง เรียงจากล่าสุด จำกัดจำนวนด้วย limit
// เตรียมไว้สำหรับทำกราฟในอนาคต ตอนนี้ยังไม่มีที่ไหนเรียกใช้
func GetSensorLogsByDeviceID(deviceID int, limit int) ([]models.SensorLog, error) {
	var logs []models.SensorLog
	// ดึงข้อมูลโดยเรียงจากเวลาล่าสุดลงไป และจำกัดจำนวนแถว (limit)
	err := DB.Where("device_id = ?", deviceID).
		Order("timestamp desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetDevicesByCoop ดึงอุปกรณ์ทั้งหมดในคอก coopID
// ถูกเรียกจาก: handlers/sensor_log.go
func GetDevicesByCoop(coopID int) ([]models.Device, error) {
	var devices []models.Device

	// ใช้คำสั่ง GORM ดึงข้อมูลอุปกรณ์ที่มี coop_id ตรงกับที่ส่งมา
	err := DB.Where("coop_id = ?", coopID).Find(&devices).Error

	return devices, err
}

// MotionAlertRow = เหตุการณ์ 1 ครั้งที่เซนเซอร์ PIR ที่วงกบประตูคอกตรวจพบการเคลื่อนไหว
// เป็นรูปแบบข้อมูลที่ส่งกลับให้หน้าบ้าน (มีชื่อคอก ชื่ออุปกรณ์ ค่า และเวลา)
// ระบุว่าเป็นเซนเซอร์ PIR จากชื่อ/ประเภทอุปกรณ์ที่มีคำว่า "pir" (แบบเดียวกับที่แอปใช้เลือกไอคอน)
type MotionAlertRow struct {
	CoopID     int       `json:"coop_id"`
	CoopName   string    `json:"coop_name"`
	DeviceID   int32     `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
}

// GetMotionAlertsForUser ดึงเหตุการณ์ตรวจพบความเคลื่อนไหวของทุกคอกของผู้ใช้ ตั้งแต่เวลา since
//
// ขั้นตอน:
//  1. ดึงคอกทั้งหมดของผู้ใช้ (GetAllCoops) ถ้าไม่มีคอก คืนรายการว่าง
//  2. หาอุปกรณ์ในคอกเหล่านั้นที่ชื่อหรือประเภทมีคำว่า pir (LOWER(...) LIKE '%pir%' ไม่สนตัวพิมพ์)
//  3. ดึง sensor_log ของอุปกรณ์เหล่านั้น ตั้งแต่ since และ value > 0 เท่านั้น
//     ทำไมต้อง value > 0: บอร์ดส่งค่า 0 (ไม่พบอะไร) มาเรื่อย ๆ ถ้าไม่กรอง ค่า 0 จะถูกนับว่า
//     "ตรวจพบ" และแจ้งเตือนจะเด้งกลับมาทันทีหลังกด "รับทราบ"
//  4. ประกอบผลลัพธ์ โดยใช้ map จับคู่ device -> คอก / ชื่อ (เร็วกว่าการ query ซ้ำทีละแถว)
//
// ถูกเรียกจาก: handlers/sensor_log.go -> GetMotionAlertsHandler (GET /api/motion-alerts)
// ใช้แสดง "ตรวจพบการเคลื่อนไหว" ในหน้าการแจ้งเตือนและหน้าข้อมูลคอกไก่
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
	// value > 0 เท่านั้น: sensor_log เป็นแถวเดียวต่อเซนเซอร์ที่ถูกเขียนทับทุกครั้งที่บอร์ดส่งค่า
	// (รวมค่า 0 = ไม่พบ) ถ้าไม่กรอง ค่า 0 จะถูกนับเป็น "ตรวจพบ" และ timestamp ที่ต่ออายุเอง
	// ทุกรอบทำให้แจ้งเตือนกลับมาทันทีหลังกดรับทราบ
	if err := DB.Where("device_id IN ? AND timestamp >= ? AND value > 0", deviceIDs, since).
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
