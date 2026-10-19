// =====================================================================================
// ไฟล์ models/sensor_log.go - โครงสร้างตาราง sensor_log (ค่าที่อ่านได้จากเซนเซอร์)
// =====================================================================================
package models

import "time"

// SensorLog ค่าจากเซนเซอร์ของอุปกรณ์ (device_id) ในคอก (coop_id) และเวลาที่ได้ค่า
// ช่องทาง MQTT (services/sensor_log.go) เก็บแค่ "ค่าล่าสุด" 1 แถวต่ออุปกรณ์ โดยเขียนทับ
// ใช้แสดงค่าปัจจุบัน (attachLatestSensorValues) และแจ้งเตือนการเคลื่อนไหวจาก PIR
type SensorLog struct {
	LogID     int32     `gorm:"primaryKey;autoIncrement;column:log_id" json:"log_id"` // 🌟 เพิ่มบรรทัดนี้
	CoopID    int32     `gorm:"column:coop_id;type:int" json:"coop_id"`
	DeviceID  int32     `gorm:"column:device_id;type:int" json:"device_id"`
	Name      string    `gorm:"column:name;type:varchar(100);index" json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`

	Device Device `gorm:"foreignKey:DeviceID;constraint:-" json:"device,omitempty"`
}

// TableName บอก GORM ว่าใช้ตาราง "sensor_log"
func (SensorLog) TableName() string {
	return "sensor_log"
}
