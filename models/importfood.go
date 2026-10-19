// =====================================================================================
// ไฟล์ models/importfood.go - โครงสร้างตาราง importfood (ประวัติการนำเข้าอาหาร)
// =====================================================================================
package models

import "time"

// ImportFood ล็อตอาหารที่นำเข้า 1 ครั้ง: ประเภท, ปริมาณ (กก.), วันที่นำเข้า, วันหมดอายุ
// เก็บเป็นประวัติ ส่วนยอดรวมปัจจุบันอยู่ที่ตาราง foodstock
type ImportFood struct {
	LotID        int       `gorm:"primaryKey;autoIncrement;column:lot_id;type:int" json:"lot_id"`
	UserID       int       `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	FoodType     string    `gorm:"column:food_type;type:varchar(20)" json:"food_type"`
	ImportVolume int       `gorm:"column:import_volume;type:int;not null" json:"import_volume"`
	ImportDate   time.Time `gorm:"column:import_date;type:datetime;not null" json:"import_date"`
	ExpiryDate   time.Time `gorm:"column:expiry_date;type:date;not null" json:"expiry_date"`
}

// TableName บอก GORM ว่าใช้ตาราง "importfood"
func (ImportFood) TableName() string {
	return "importfood"
}
