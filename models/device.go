package models

import "time"

// Device represents farm equipment/sensors
type Device struct {
	DeviceID      int       `gorm:"primaryKey;column:device_id;type:int" json:"device_id"`
	CoopID        int       `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
	SlotIndex     int       `gorm:"column:slot_index;type:int;not null" json:"slot_index"` // 🌟 1. เพิ่มฟิลด์เก็บเลขช่อง (0-20)
	Name          string    `gorm:"column:name;type:varchar(100)" json:"name"`
	Icon          string    `gorm:"column:icon;type:varchar(255)" json:"icon"`             // 🌟 2. เพิ่มฟิลด์เก็บรูปไอคอนอุปกรณ์
	DeviceType    string    `gorm:"column:device_type;type:varchar(50)" json:"device_type"`
	CurrentStatus string    `gorm:"column:current_status;type:varchar(20);default:'Offline'" json:"current_status"`
	LastUpdate    time.Time `gorm:"column:last_update" json:"last_update"`

	// 🌟 เติม constraint:- เพื่อห้ามไม่ให้ GORM สร้าง Foreign Key ย้อนกลับแบบมั่วๆ
	Coop       Coop        `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
	// เติม constraint:- เข้าไปเพิ่มครับ
   // ต้องมี constraint:- แบบนี้เป๊ะๆ นะครับ
    SensorLogs []SensorLog `gorm:"foreignKey:DeviceID;constraint:-" json:"sensor_logs,omitempty"`
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