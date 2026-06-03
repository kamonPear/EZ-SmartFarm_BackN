package models

// SensorLog represents sensor data records
type SensorLog struct {
	LogID    int `gorm:"primaryKey;autoIncrement;column:log_id;type:int" json:"log_id"`
	DeviceID int `gorm:"column:device_id"`
	// Relations
	Device Device `gorm:"foreignKey:DeviceID;constraint:-" json:"device,omitempty"`
}

// TableName specifies the table name for SensorLog model
func (SensorLog) TableName() string {
	return "sensor_log"
}
