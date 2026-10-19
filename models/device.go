// =====================================================================================
// ไฟล์ models/device.go - โครงสร้างตาราง device (อุปกรณ์ในคอก) และ JSON ที่เกี่ยวกับผังอุปกรณ์
// =====================================================================================
package models

import "time"

// Device อุปกรณ์ 1 ตัวที่ติดตั้งในคอก (ตาราง device)
//  - CoopID + SlotIndex : ตัวระบุอุปกรณ์ บอร์ดส่งค่ามาพร้อม 2 ค่านี้ (ห้ามซ้ำ: uq_device_coop_slot)
//  - Name / Icon        : ชื่อและไอคอน (คัดลอกมาจาก device_type ตอนวาง) Icon เป็น text เพราะ SVG ยาว
//  - DeviceType         : "Sensor" (วัดค่า) หรือ "Actuator" (พัดลม/หลอดไฟ)
//  - CurrentStatus      : Online / Offline (อัปเดตโดย services/sensor_log.go และ StartOfflineChecker)
//  - LastUpdate         : เวลาที่ได้ข้อมูลล่าสุด
//  - PosX / PosY        : ตำแหน่งบนผังคอก (เปอร์เซ็นต์ 0-100) nil = อุปกรณ์รุ่นเก่า ใช้ตำแหน่งจาก slot แทน
//  - Value              : ค่าเซนเซอร์ล่าสุด gorm:"-" = ไม่ใช่คอลัมน์ เติมทีหลังด้วย attachLatestSensorValues
// constraint:- = ไม่ให้ GORM สร้าง Foreign Key เอง (จัดการเองใน migration)
type Device struct {
	DeviceID  int32  `gorm:"primaryKey;column:device_id;type:int" json:"device_id"`
	CoopID    int32  `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
	NameCoop  string `gorm:"column:name_coop;type:varchar(100);index" json:"name_coop"`
	SlotIndex int32  `gorm:"column:slot_index;type:int;not null" json:"slot_index"`
	Name      string `gorm:"column:name;type:varchar(100);index" json:"name"`
	// text ไม่ใช่ varchar(255) เพราะไอคอนเก็บเป็น SVG data URI (500+ ตัวอักษร) คัด
	// ลอกมาจาก device_type.icon ตอนวางอุปกรณ์ลงคอก - varchar(255) เดิมแคบเกินไป
	// ทำให้ MySQL strict mode insert ไม่ผ่านแบบเงียบๆ (เจอบั๊กเดียวกันกับ
	// device_type.icon มาแล้ว ดู ensureColumnIsText ใน migration.go)
	Icon          string    `gorm:"column:icon;type:text" json:"icon"`
	DeviceType    string    `gorm:"column:device_type;type:varchar(50)" json:"device_type"`
	CurrentStatus string    `gorm:"column:current_status;type:varchar(20);default:'Offline'" json:"current_status"`
	LastUpdate    time.Time `gorm:"column:last_update" json:"last_update"`

	// จุดที่ผู้ใช้วางอุปกรณ์จริงบนผังคอก (เปอร์เซ็นต์ 0-100 ของแคนวาส) เก็บแยกจาก
	// slot_index ซึ่งยังเป็นตัวระบุอุปกรณ์สำหรับจับคู่ข้อมูลจากบอร์ด (coop_id + slot_index)
	// เหมือนเดิม - NULL = อุปกรณ์ที่วางไว้ก่อนมีคอลัมน์นี้ ให้ client ใช้ตำแหน่งกริดจาก slot_index แทน
	PosX *float64 `gorm:"column:pos_x;type:double" json:"pos_x"`
	PosY *float64 `gorm:"column:pos_y;type:double" json:"pos_y"`

	Coop       Coop        `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
	SensorLogs []SensorLog `gorm:"foreignKey:DeviceID;constraint:-" json:"sensor_logs,omitempty"`

	// 🔥 เพิ่มบรรทัดนี้เข้ามาข้างล่างสุดของ Struct ครับ
	// ใช้ gorm:"-" เพื่อบอกว่า ฟิลด์นี้ไม่ได้เป็นคอลัมน์จริงๆ ในตาราง device แต่มันจะเกิดจากการเอาข้อมูลมาแปะทีหลัง
	Value *float64 `gorm:"-" json:"value"`
}

// TableName บอก GORM ว่าใช้ตาราง "device"
func (Device) TableName() string {
	return "device"
}

// =================================================================
// 🌟 3. วางโครงสร้างสำหรับรับ Request Layout จาก Angular ไว้ตรงนี้ได้เลย
// =================================================================

// DevicePayload อุปกรณ์ในช่อง 1 ช่องที่หน้าเว็บส่งมาตอนบันทึกผัง
//  - SlotIndex : ส่งมาเฉพาะอุปกรณ์เดิม (ให้ backend รู้ว่าเป็นตัวเดิม คงเลขเดิม)
//                ไม่ส่ง (nil) = อุปกรณ์ใหม่ backend ออกเลขถัดไปให้
type DevicePayload struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
	// เลขอุปกรณ์ที่ backend ออกให้แล้ว (device.slot_index) - ส่งมาเฉพาะอุปกรณ์ที่มีอยู่แล้วในคอก ให้ backend
	// รู้ว่าเป็นตัวเดิม (คงเลขเดิม/อัปเดตตำแหน่ง) ถ้าไม่ส่ง = อุปกรณ์ใหม่ backend จะออกเลขถัดไปให้เอง
	SlotIndex *int32 `json:"slot_index"`
}

// SlotPayload 1 ช่องบนผังคอก: Device เป็น pointer เพราะช่องว่างจะส่งมาเป็น null
// X / Y = ตำแหน่งบนแคนวาส (เปอร์เซ็นต์) หน้าเว็บรุ่นเก่าที่ไม่ส่งจะได้ nil
// ใช้ใน: database.SaveCoopLayout
type SlotPayload struct {
	ID     int            `json:"id"`
	Device *DevicePayload `json:"device"` // ใช้ Pointer เพื่อรองรับค่า null เวลาช่องนั้นว่าง
	Status string         `json:"status"`
	// ตำแหน่งที่วางจริงบนแคนวาส (เปอร์เซ็นต์) - ไม่บังคับ client เก่าที่ไม่ส่งมาจะได้ NULL
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
}

// LayoutPayload body ทั้งก้อนของ POST /api/coops/layout = รายการทุกช่องในคอก
type LayoutPayload struct {
	Slots []SlotPayload `json:"slots"`
}

// ไฟล์: models/device.go (หรือไฟล์ที่คุณตั้งชื่อไว้)

// ... (โค้ด Device struct เดิมของคุณ) ...

// ArduinoPayload JSON ที่บอร์ดส่งมาทาง HTTP (POST /api/sensor/upload ใน handlers/controllers.go)
// จับคู่อุปกรณ์ด้วย coop_id + slot_index (ชื่อใช้แค่ log)
// ทางหลักที่ใช้จริงคือ MQTT ซึ่งใช้ services.SensorPayload
type ArduinoPayload struct {
	CoopID      int     `json:"coop_id"`
	SlotIndex   int     `json:"slot_index"`  // ตำแหน่งช่องที่วางอุปกรณ์ไว้ (ใช้จับคู่อุปกรณ์แทนชื่อ เพราะชื่อซ้ำกันได้)
	DeviceName  string  `json:"device_name"` // ชื่อเซ็นเซอร์ เช่น MQ-135 (ใช้แค่ log/debug ไม่ใช้จับคู่แล้ว)
	SensorValue float64 `json:"sensor_value"`
}
