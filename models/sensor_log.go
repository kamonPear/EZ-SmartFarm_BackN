package models

import "time"

// SensorLog represents sensor data records
type SensorLog struct {
	LogID     int       `gorm:"primaryKey;autoIncrement;column:log_id" json:"log_id"`
	DeviceID  int       `gorm:"column:device_id;index" json:"device_id"`
	Value     float64   `gorm:"column:value;type:decimal(10,2)" json:"value"`
	Timestamp time.Time `gorm:"column:timestamp;autoCreateTime:milli" json:"timestamp"`

	// Relations
	Device Device `gorm:"foreignKey:DeviceID;references:DeviceID" json:"device,omitempty"`
}

// TableName specifies the table name for SensorLog model
func (SensorLog) TableName() string {
	return "sensor_log"
}
