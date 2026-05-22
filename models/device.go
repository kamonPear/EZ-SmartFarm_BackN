package models

import "time"

// Device represents farm equipment/sensors
type Device struct {
	DeviceID      int       `gorm:"primaryKey;column:device_id;type:int" json:"device_id"`
	CoopID        int       `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
	Name          string    `gorm:"column:name;type:varchar(100)" json:"name"`
	DeviceType    string    `gorm:"column:device_type;type:varchar(50)" json:"device_type"`
	CurrentStatus string    `gorm:"column:current_status;type:varchar(20);default:'Offline'" json:"current_status"`
	LastUpdate    time.Time `gorm:"column:last_update" json:"last_update"`

	// Relations
	Coop       Coop        `gorm:"foreignKey:CoopID;references:CoopID" json:"coop,omitempty"`
	SensorLogs []SensorLog `gorm:"foreignKey:DeviceID;references:DeviceID" json:"sensor_logs,omitempty"`
}

// TableName specifies the table name for Device model
func (Device) TableName() string {
	return "device"
}
