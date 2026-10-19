// =====================================================================================
// ไฟล์ models/coop.go - โครงสร้างตาราง coop (คอกไก่)
// =====================================================================================
//
// โฟลเดอร์ models/ เก็บ "รูปร่างข้อมูล" (struct) ของทั้งระบบ แบ่งเป็น 2 แบบ:
//  - model ของตาราง (เช่น Coop) : 1 struct = 1 ตาราง, 1 field = 1 คอลัมน์
//  - request (ไฟล์ *_request.go)  : รูปแบบ JSON ที่แอป/เว็บส่งเข้ามา
//
// อ่าน tag ใน struct ยังไง:
//  - gorm:"column:coop_id;type:int" = บอก GORM ว่า field นี้คือคอลัมน์ไหน ชนิดอะไร (ใช้ตอน AutoMigrate/query)
//  - json:"coop_id"                = ชื่อ field ตอนแปลงเป็น JSON ส่งให้แอป/เว็บ
//  - json:"-"                      = ไม่ส่งออกทาง JSON เลย
//  - omitempty                      = ถ้าว่างไม่ต้องใส่ใน JSON
package models

import "time"

// Coop คอกไก่ 1 คอก (ตาราง coop)
//  - UserID           : เจ้าของคอก ใช้แยกข้อมูลของแต่ละผู้ใช้
//  - NameCoop         : ชื่อคอก ห้ามซ้ำภายในผู้ใช้เดียวกัน (unique key uq_coop_user_name ใน migration)
//  - DateAdoptAnimals : วันที่รับไก่เข้าเลี้ยง
//  - Amount           : จำนวนไก่ ใช้คำนวณอาหารต่อวัน
//  - Birthday         : วันเกิดไก่ ใช้คำนวณอายุ -> วัคซีน, ประเภทอาหาร, เริ่มเก็บไข่ได้เมื่อไร
//  - PosX / PosY      : ตำแหน่งบนผังฟาร์ม (เป็น pointer เพราะ nil = ยังไม่ได้วาง)
//  - NextDeviceNo     : เลขอุปกรณ์ตัวถัดไปของคอก (ไม่ส่งออก JSON)
//  - Devices/Eggs/Health/VaccineHistory : ข้อมูลลูก โหลดด้วย Preload (ไม่ใช่คอลัมน์)
type Coop struct {
	CoopID           int       `gorm:"primaryKey;autoIncrement;column:coop_id;type:int" json:"coop_id"`
	UserID           int       `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	NameCoop         string    `gorm:"column:name_coop;type:varchar(100)" json:"name_coop"` // ไม่ซ้ำ "ต่อผู้ใช้" - ดู uq_coop_user_name_coop ใน migration (ไม่ใช่ unique ทั้งระบบ)
	DateAdoptAnimals time.Time `gorm:"column:date_adopt_animals" json:"date_adopt_animals"`
	Amount           int       `gorm:"column:amount" json:"amount"`
	Birthday         time.Time `gorm:"index:idx_coop_birthday" json:"birthday"`
	Note             string    `gorm:"column:note;type:text" json:"note"`

	// ตำแหน่งบนผังฟาร์ม (nil = ยังไม่ได้ลากไปวางบนผัง) - ตั้งค่าผ่าน
	// PUT /api/coops/positions เท่านั้น ไม่ใช่ผ่าน UpdateCoop ทั่วไป
	PosX *float64 `gorm:"column:pos_x" json:"pos_x"`
	PosY *float64 `gorm:"column:pos_y" json:"pos_y"`

	// เลขอุปกรณ์ (device.slot_index) ที่จะให้ตัวถัดไปของคอกนี้ - เริ่ม 1 ต่อคอก วิ่งขึ้นเรื่อยๆ ไม่ใช้เลขที่เคย
	// ออกไปแล้วซ้ำแม้อุปกรณ์นั้นจะถูกลบ (กันบอร์ดที่ยังส่งเลขเก่าเข้าผิดอุปกรณ์) ไม่ส่งออกทาง JSON
	NextDeviceNo int `gorm:"column:next_device_no;type:int;not null;default:1" json:"-"`

	// Relations (แก้โดยการลบ references:CoopID ออก)
	Devices        []Device         `gorm:"foreignKey:CoopID" json:"devices,omitempty"`
	Eggs           []Egg            `gorm:"foreignKey:CoopID" json:"eggs,omitempty"`
	Health         []Health         `gorm:"foreignKey:CoopID" json:"health,omitempty"`
	VaccineHistory []VaccineHistory `gorm:"foreignKey:CoopID" json:"vaccine_history,omitempty"`
}

// TableName บอก GORM ว่า struct นี้ใช้ตารางชื่อ "coop"
// (ถ้าไม่บอก GORM จะเดาชื่อเป็นพหูพจน์ "coops" ซึ่งไม่ตรงกับฐานข้อมูลจริง)
func (Coop) TableName() string {
	return "coop"
}
