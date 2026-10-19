// =====================================================================================
// ไฟล์ models/vaccine.go - โครงสร้างตาราง vaccine ("ประเภท/เกณฑ์" วัคซีน-ยา)
// =====================================================================================
package models

import "time"

// Vaccine ประเภทวัคซีน/ยา 1 ตัว: ชื่อ, วิธีให้, หมายเหตุ, ช่วงอายุที่ควรให้ (min_age_days - max_age_days วัน)
// ใช้ร่วมกับทุกคอกในฟาร์ม ไม่ผูกกับคอกไหน (ผูกกับผู้ใช้ผ่าน user_id)
// ประวัติการให้จริงแยกไปอยู่ที่ VaccineHistory
// ทำไมแยก 2 ตาราง: ยา 1 ตัวใช้ซ้ำได้หลายคอก ถ้ารวมไว้ตารางเดียว ผู้ใช้จะสับสนว่าแถวไหนเป็นเกณฑ์
// แถวไหนเป็นประวัติ (รุ่นเก่าเคยรวมกันแล้วเกิดปัญหานี้)
type Vaccine struct {
	VaccineID  int    `gorm:"primaryKey;column:vaccine_id" json:"vaccine_id"`
	UserID     int    `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	Name       string `gorm:"column:name_vaccine;type:varchar(50);not null" json:"name"`
	Method     string `gorm:"column:method;type:varchar(100);not null" json:"method"`
	Note       string `gorm:"column:note;type:varchar(100)" json:"note"`
	MinAgeDays int    `gorm:"column:min_age_days;type:int;not null" json:"min_age_days"`
	MaxAgeDays int    `gorm:"column:max_age_days;type:int;not null" json:"max_age_days"`
}

// TableName บอก GORM ว่าใช้ตาราง "vaccine"
func (Vaccine) TableName() string {
	return "vaccine"
}

// CalculateChickenAge คำนวณอายุไก่ ณ ตอนนี้ เป็นจำนวนวัน = (ตอนนี้ - วันเกิด) / 24 ชั่วโมง
// ไม่มีวันเกิด หรือวันเกิดอยู่ในอนาคต -> คืน 0 (กันค่าติดลบ)
// ถูกเรียกจาก: handlers/vaccine.go (แนะนำวัคซีน และหาคอกที่ต้องให้วัคซีน)
func CalculateChickenAge(birthday time.Time) int {
	if birthday.IsZero() {
		return 0
	}
	ageInDays := int(time.Since(birthday).Hours() / 24)
	if ageInDays < 0 {
		return 0
	}
	return ageInDays
}
