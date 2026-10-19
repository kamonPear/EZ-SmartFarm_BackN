package database

import (
	"strings"
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

// fanTrigger คือเหตุที่ทำให้พัดลมของคอกหนึ่ง "ควรทำงาน" ตอนนี้
type fanTrigger struct {
	cause     string
	value     float64
	threshold float64
}

// idleFansWhenHot คืนพัดลมที่ "ควรทำงานแต่ไม่ทำงาน" - ตรงกับลอจิกบนบอร์ด (พัดลมทุกตัวในคอก
// ติดเมื่ออุณหภูมิ >= ค่ามาตรฐานฟาร์ม หรือแอมโมเนีย >= ค่ามาตรฐานแอมโมเนีย) คอกที่กำลังส่ง
// ข้อมูลอยู่และค่าล่าสุด (ภายใน coopActiveWindow) ถึงเกณฑ์ แต่พัดลมยังเป็น Offline (บอร์ดส่ง
// 0 เมื่อรีเลย์ดับ หรือสั่งติดแล้วกระแสไม่ไหล) เกณฑ์ที่ยังไม่ตั้ง (0) = ข้าม
func idleFansWhenHot(userID int, coopIDs []int, coopNameByID map[int]string) ([]OfflineSensorRow, error) {
	threshold, err := GetFarmThreshold(userID)
	if err != nil {
		return nil, err
	}
	if threshold.Temperature <= 0 && threshold.Ammonia <= 0 {
		return nil, nil
	}

	var probes []models.Device
	if err := DB.Where("coop_id IN ? AND (UPPER(name) LIKE ? OR UPPER(name) LIKE ?)", coopIDs, "DHT%", "MQ%").
		Find(&probes).Error; err != nil {
		return nil, err
	}
	if len(probes) == 0 {
		return nil, nil
	}
	probeByID := make(map[int32]models.Device, len(probes))
	probeIDs := make([]int32, len(probes))
	for i, d := range probes {
		probeByID[d.DeviceID] = d
		probeIDs[i] = d.DeviceID
	}

	var logs []models.SensorLog
	if err := DB.Where("device_id IN ? AND timestamp >= ?", probeIDs, time.Now().Add(-coopActiveWindow)).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	triggers := make(map[int32]fanTrigger)
	for _, l := range logs {
		d := probeByID[l.DeviceID]
		if strings.HasPrefix(strings.ToUpper(d.Name), "DHT") {
			if threshold.Temperature > 0 && l.Value >= threshold.Temperature {
				triggers[d.CoopID] = fanTrigger{"temperature", l.Value, threshold.Temperature}
			}
		} else if _, hasTemp := triggers[d.CoopID]; !hasTemp {
			if threshold.Ammonia > 0 && l.Value >= threshold.Ammonia {
				triggers[d.CoopID] = fanTrigger{"ammonia", l.Value, threshold.Ammonia}
			}
		}
	}
	if len(triggers) == 0 {
		return nil, nil
	}
	hotIDs := make([]int32, 0, len(triggers))
	for id := range triggers {
		hotIDs = append(hotIDs, id)
	}

	var fans []models.Device
	if err := DB.Where(
		"coop_id IN ? AND device_type = ? AND name LIKE ? AND current_status = ?",
		hotIDs, "Actuator", "พัดลม%", "Offline",
	).Order("coop_id, slot_index").Find(&fans).Error; err != nil {
		return nil, err
	}

	rows := make([]OfflineSensorRow, 0, len(fans))
	for _, f := range fans {
		tr := triggers[f.CoopID]
		value, limit := tr.value, tr.threshold
		rows = append(rows, OfflineSensorRow{
			Kind:       "fan",
			Cause:      tr.cause,
			Value:      &value,
			Threshold:  &limit,
			CoopID:     int(f.CoopID),
			CoopName:   coopNameByID[int(f.CoopID)],
			DeviceID:   f.DeviceID,
			DeviceName: f.Name,
			SlotIndex:  f.SlotIndex,
			LastUpdate: f.LastUpdate,
		})
	}
	return rows, nil
}

// idleLampsWhenCold คืนหลอดไฟที่ "ควรทำงานแต่ไม่ทำงาน" - คอกที่กำลังส่งข้อมูลอยู่และอุณหภูมิล่าสุด
// (DHT ภายใน coopActiveWindow) ต่ำกว่าค่ามาตรฐานอุณหภูมิฟาร์ม (ต้องกกให้อุ่น) แต่หลอดไฟยังเป็น
// Offline (บอร์ดส่ง 0 เมื่อรีเลย์ดับ หรือสั่งติดแล้วกระแสไม่ไหล) เกณฑ์ที่ยังไม่ตั้ง (0) = ข้าม
func idleLampsWhenCold(userID int, coopIDs []int, coopNameByID map[int]string) ([]OfflineSensorRow, error) {
	threshold, err := GetFarmThreshold(userID)
	if err != nil {
		return nil, err
	}
	if threshold.Temperature <= 0 {
		return nil, nil
	}

	var probes []models.Device
	if err := DB.Where("coop_id IN ? AND UPPER(name) LIKE ?", coopIDs, "DHT%").
		Find(&probes).Error; err != nil {
		return nil, err
	}
	if len(probes) == 0 {
		return nil, nil
	}
	probeByID := make(map[int32]models.Device, len(probes))
	probeIDs := make([]int32, len(probes))
	for i, d := range probes {
		probeByID[d.DeviceID] = d
		probeIDs[i] = d.DeviceID
	}

	var logs []models.SensorLog
	if err := DB.Where("device_id IN ? AND timestamp >= ?", probeIDs, time.Now().Add(-coopActiveWindow)).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	coldTemp := make(map[int32]float64)
	for _, l := range logs {
		if l.Value < threshold.Temperature {
			coldTemp[probeByID[l.DeviceID].CoopID] = l.Value
		}
	}
	if len(coldTemp) == 0 {
		return nil, nil
	}
	coldIDs := make([]int32, 0, len(coldTemp))
	for id := range coldTemp {
		coldIDs = append(coldIDs, id)
	}

	var lamps []models.Device
	if err := DB.Where(
		"coop_id IN ? AND device_type = ? AND name LIKE ? AND current_status = ?",
		coldIDs, "Actuator", "หลอดไฟ%", "Offline",
	).Order("coop_id, slot_index").Find(&lamps).Error; err != nil {
		return nil, err
	}

	rows := make([]OfflineSensorRow, 0, len(lamps))
	for _, l := range lamps {
		value, limit := coldTemp[l.CoopID], threshold.Temperature
		rows = append(rows, OfflineSensorRow{
			Kind:       "lamp",
			Cause:      "temperature",
			Value:      &value,
			Threshold:  &limit,
			CoopID:     int(l.CoopID),
			CoopName:   coopNameByID[int(l.CoopID)],
			DeviceID:   l.DeviceID,
			DeviceName: l.Name,
			SlotIndex:  l.SlotIndex,
			LastUpdate: l.LastUpdate,
		})
	}
	return rows, nil
}

// coopActiveWindow คือช่วงเวลาที่ถือว่าคอกยัง "ส่งข้อมูลอยู่" - เผื่อไว้มากกว่าเกณฑ์
// Offline ของ services.StartOfflineChecker (30 วินาที) เล็กน้อย กันแจ้งเตือนกะพริบ
const coopActiveWindow = 60 * time.Second

// OfflineSensorRow คือเซนเซอร์ 1 ตัวที่ไม่ทำงาน (สถานะ Offline = เงียบเกิน
// offlineAfterSeconds ตาม services.StartOfflineChecker) หรือพัดลม 1 ตัวที่ควรทำงานแต่ไม่ทำงาน
// - ใช้ทำแจ้งเตือนทั้งบนเว็บและแอป
//
// Kind = "sensor" (เซนเซอร์เงียบ) หรือ "lamp" (อุณหภูมิต่ำกว่าเกณฑ์ฟาร์มแต่หลอดไฟตัวนี้ไม่ทำงาน
// - Cause = "temperature") หรือ "fan" (ค่าถึงเกณฑ์ฟาร์มแล้วแต่พัดลมตัวนี้ไม่ทำงาน
// - Cause = "temperature" | "ammonia", Value/Threshold ใช้แสดงในข้อความ มีเฉพาะ kind = "fan")
type OfflineSensorRow struct {
	Kind       string    `json:"kind"`
	Cause      string    `json:"cause,omitempty"`
	Value      *float64  `json:"value,omitempty"`
	Threshold  *float64  `json:"threshold,omitempty"`
	CoopID     int       `json:"coop_id"`
	CoopName   string    `json:"coop_name"`
	DeviceID   int32     `json:"device_id"`
	DeviceName string    `json:"device_name"`
	SlotIndex  int32     `json:"slot_index"`
	LastUpdate time.Time `json:"last_update"`
}

// GetOfflineSensorsForUser คืนเซนเซอร์ที่ Offline ของทุกคอกที่ userID เป็นเจ้าของ
// เฉพาะ device_type = 'Sensor' - อุปกรณ์สั่งงาน (พัดลม/หลอดไฟ) จะเป็น Offline เมื่อ
// "ปิดอยู่" ตามปกติ ไม่ใช่อาการเสีย จึงไม่นับ (พัดลมดูแยกใน idleFansWhenHot) และเฉพาะ
// คอกที่กำลังส่งข้อมูลมาจาก MQTT อยู่
func GetOfflineSensorsForUser(userID int) ([]OfflineSensorRow, error) {
	coops, err := GetAllCoops(userID)
	if err != nil {
		return nil, err
	}
	if len(coops) == 0 {
		return []OfflineSensorRow{}, nil
	}

	coopIDs := make([]int, len(coops))
	coopNameByID := make(map[int]string, len(coops))
	for i, c := range coops {
		coopIDs[i] = c.CoopID
		coopNameByID[c.CoopID] = c.NameCoop
	}

	// เช็คเฉพาะคอกที่ "กำลังส่งข้อมูลอยู่ตอนนี้" - มีอุปกรณ์อย่างน้อย 1 ตัวในคอกที่
	// last_update ภายใน coopActiveWindow (MQTT อัปเดต last_update ทุกครั้งที่รับข้อความ
	// ไม่ว่าจะเป็นเซนเซอร์หรืออุปกรณ์สั่งงาน) คอกที่เงียบทั้งคอก (บอร์ดปิด/ยังไม่ติดตั้ง)
	// จึงไม่แจ้ง เพราะเป็นเรื่องบอร์ด ไม่ใช่เซนเซอร์ตัวใดตัวหนึ่งเสีย
	var activeCoopIDs []int
	if err := DB.Model(&models.Device{}).
		Where("coop_id IN ? AND last_update >= ?", coopIDs, time.Now().Add(-coopActiveWindow)).
		Distinct().
		Pluck("coop_id", &activeCoopIDs).Error; err != nil {
		return nil, err
	}
	if len(activeCoopIDs) == 0 {
		return []OfflineSensorRow{}, nil
	}

	var devices []models.Device
	if err := DB.Where(
		"coop_id IN ? AND device_type = ? AND current_status = ?",
		activeCoopIDs, "Sensor", "Offline",
	).Order("coop_id, slot_index").Find(&devices).Error; err != nil {
		return nil, err
	}

	result := make([]OfflineSensorRow, 0, len(devices))
	for _, d := range devices {
		result = append(result, OfflineSensorRow{
			Kind:       "sensor",
			CoopID:     int(d.CoopID),
			CoopName:   coopNameByID[int(d.CoopID)],
			DeviceID:   d.DeviceID,
			DeviceName: d.Name,
			SlotIndex:  d.SlotIndex,
			LastUpdate: d.LastUpdate,
		})
	}

	fanRows, err := idleFansWhenHot(userID, activeCoopIDs, coopNameByID)
	if err != nil {
		return nil, err
	}
	lampRows, err := idleLampsWhenCold(userID, activeCoopIDs, coopNameByID)
	if err != nil {
		return nil, err
	}
	return append(append(result, fanRows...), lampRows...), nil
}
