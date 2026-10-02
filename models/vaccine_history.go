package models

import "time"

// VaccineHistory คือประวัติการให้วัคซีน/ยาจริงกับคอกใดคอกหนึ่ง แยกออกจากตาราง vaccine
// (ซึ่งเก็บแค่ "เกณฑ์" ของยาแต่ละตัว ไม่ผูกกับคอกไหน) - ทุกแถวในตารางนี้คือเหตุการณ์
// "ให้จริง" แล้วเสมอ จึงไม่ต้องใช้ pointer field เหมือนตอนที่เคยรวมสองแนวคิดไว้
// ตารางเดียวกัน (coop_id/birthday/record_date เป็นค่าบังคับมีเสมอ ไม่มี NULL)
type VaccineHistory struct {
	VaccineID int    `gorm:"primaryKey;column:id" json:"vaccine_id"`
	CoopID    int    `gorm:"column:coop_id;type:int;size:32;not null;index" json:"coop_id"`
	NameCoop  string `gorm:"column:name_coop;type:varchar(100);index" json:"name_coop"`

	Birthday time.Time `gorm:"column:birthday;type:date" json:"birthday"`

	Name           string    `gorm:"column:name_vaccine;type:varchar(50);not null" json:"name"`
	Method         string    `gorm:"column:method;type:varchar(100);not null" json:"method"`
	Note           string    `gorm:"column:note;type:varchar(100)" json:"note"`
	RecordDate     time.Time `gorm:"column:record_date;type:date;not null" json:"record_date"`
	RecommendedAge string    `gorm:"column:recommended_age;type:varchar(20)" json:"recommended_age"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

func (VaccineHistory) TableName() string {
	return "vaccine_history"
}
