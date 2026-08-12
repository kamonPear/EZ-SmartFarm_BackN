package models

import "time"

// Device represents farm equipment/sensors
// Device represents farm equipment/sensors
type Device struct {
    DeviceID      int32       `gorm:"primaryKey;column:device_id;type:int" json:"device_id"`
    CoopID        int32       `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
    NameCoop      string      `gorm:"column:name_coop;type:varchar(100);index" json:"name_coop"`
    SlotIndex     int32       `gorm:"column:slot_index;type:int;not null" json:"slot_index"`
    Name          string      `gorm:"column:name;type:varchar(100);unique" json:"name"`
    Icon          string      `gorm:"column:icon;type:varchar(255)" json:"icon"`            
    DeviceType    string      `gorm:"column:device_type;type:varchar(50)" json:"device_type"`
    CurrentStatus string      `gorm:"column:current_status;type:varchar(20);default:'Offline'" json:"current_status"`
    LastUpdate    time.Time   `gorm:"column:last_update" json:"last_update"`

    Coop          Coop        `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
    SensorLogs    []SensorLog `gorm:"foreignKey:DeviceID;constraint:-" json:"sensor_logs,omitempty"`

    // 🔥 เพิ่มบรรทัดนี้เข้ามาข้างล่างสุดของ Struct ครับ
    // ใช้ gorm:"-" เพื่อบอกว่า ฟิลด์นี้ไม่ได้เป็นคอลัมน์จริงๆ ในตาราง device แต่มันจะเกิดจากการเอาข้อมูลมาแปะทีหลัง
    Value         *float64    `gorm:"-" json:"value"` 

    
}

// TableName specifies the table name for Device model
func (Device) TableName() string {
	return "device"
}

// =================================================================
// 🌟 3. วางโครงสร้างสำหรับรับ Request Layout จาก Angular ไว้ตรงนี้ได้เลย
// =================================================================

type DevicePayload struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type SlotPayload struct {
	ID     int            `json:"id"`
	Device *DevicePayload `json:"device"` // ใช้ Pointer เพื่อรองรับค่า null เวลาช่องนั้นว่าง
	Status string         `json:"status"`
}

type LayoutPayload struct {
	Slots []SlotPayload `json:"slots"`
}

// ไฟล์: models/device.go (หรือไฟล์ที่คุณตั้งชื่อไว้)

// ... (โค้ด Device struct เดิมของคุณ) ...

// เพิ่ม Struct นี้เข้าไป เพื่อใช้รับ JSON จาก Arduino
type ArduinoPayload struct {
	CoopID      int     `json:"coop_id"`
	DeviceName  string  `json:"device_name"` // ชื่อเซ็นเซอร์ เช่น MQ-135
	SensorValue float64 `json:"sensor_value"`
}

