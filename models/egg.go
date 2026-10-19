// =====================================================================================
// ไฟล์ models/egg.go - โครงสร้างตาราง egg (บันทึกการเก็บไข่)
// =====================================================================================
package models

import "time"

// Egg บันทึกการเก็บไข่ 1 ครั้ง: คอก, วันที่เก็บ, จำนวนไข่ (ฟอง), หมายเหตุ
// คอกเดียวกันบันทึกซ้ำวันเดียวกันไม่ได้ (unique coop_id + date_collect_egg)
// Coop = ข้อมูลคอก โหลดด้วย Preload เพื่อให้แอปแสดงชื่อคอกได้
type Egg struct {
	EggID          int       `gorm:"primaryKey;autoIncrement;column:egg_id;type:int" json:"egg_id"`
	CoopID         int       `gorm:"column:coop_id;not null;type:int;size:32" json:"coop_id"`
	NameCoop       string    `gorm:"column:name_coop;type:varchar(100);index" json:"name_coop"`
	DateCollectEgg time.Time `gorm:"column:date_collect_egg" json:"date_collect_egg"`
	NumberEgg      int       `gorm:"column:number_egg" json:"number_egg"`
	Note           string    `gorm:"column:note;type:text" json:"note"`

	// Relations
	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

// TableName บอก GORM ว่าใช้ตาราง "egg"
func (Egg) TableName() string {
	return "egg"
}
