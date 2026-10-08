package models

import "time"

// Coop represents the chicken coop/farm information
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

	// Relations (แก้โดยการลบ references:CoopID ออก)
	Devices        []Device         `gorm:"foreignKey:CoopID" json:"devices,omitempty"`
	Eggs           []Egg            `gorm:"foreignKey:CoopID" json:"eggs,omitempty"`
	Health         []Health         `gorm:"foreignKey:CoopID" json:"health,omitempty"`
	VaccineHistory []VaccineHistory `gorm:"foreignKey:CoopID" json:"vaccine_history,omitempty"`
}

// TableName specifies the table name for Coop model
func (Coop) TableName() string {
	return "coop"
}
