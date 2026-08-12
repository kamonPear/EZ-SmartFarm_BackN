package models

import "time"

type SensorLog struct {
    LogID     int32     `gorm:"primaryKey;autoIncrement;column:log_id" json:"log_id"` // 🌟 เพิ่มบรรทัดนี้
    CoopID    int32     `gorm:"column:coop_id;type:int" json:"coop_id"`
    DeviceID  int32     `gorm:"column:device_id;type:int" json:"device_id"`
    Name      string    `gorm:"column:name;type:varchar(100);index" json:"name"`
    Value     float64   `json:"value"`
    Timestamp time.Time `json:"timestamp"`

    Device Device `gorm:"foreignKey:DeviceID;constraint:-" json:"device,omitempty"`
}

func (SensorLog) TableName() string {
    return "sensor_log"
}