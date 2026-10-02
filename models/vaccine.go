package models

import "time"

// Vaccine คือ "ประเภท/เกณฑ์" ของยา-วัคซีนแต่ละตัว (ชื่อ, วิธีให้, ช่วงอายุที่ควรให้)
// ใช้ร่วมกันได้กับทุกคอก ไม่ผูกกับคอกไหนทั้งนั้น - ประวัติการให้จริงแต่ละคอกแยกไปอยู่
// ตาราง VaccineHistory ต่างหาก (แยกเป็น 2 ตารางเพราะยา 1 ตัวใช้ซ้ำได้กับหลายคอก ถ้า
// รวมไว้ตารางเดียวจะงงว่าแถวไหนเป็นเกณฑ์ แถวไหนเป็นประวัติให้จริง - ของเดิมเคยรวม
// ไว้ตารางเดียวกันแล้วสับสนตามที่ผู้ใช้สะท้อนมา จึงแยกกลับเป็น 2 ตาราง)
type Vaccine struct {
	VaccineID  int    `gorm:"primaryKey;column:vaccine_id" json:"vaccine_id"`
	Name       string `gorm:"column:name_vaccine;type:varchar(50);not null" json:"name"`
	Method     string `gorm:"column:method;type:varchar(100);not null" json:"method"`
	Note       string `gorm:"column:note;type:varchar(100)" json:"note"`
	MinAgeDays int    `gorm:"column:min_age_days;type:int;not null" json:"min_age_days"`
	MaxAgeDays int    `gorm:"column:max_age_days;type:int;not null" json:"max_age_days"`
}

func (Vaccine) TableName() string {
	return "vaccine"
}

// CalculateChickenAge ฟังก์ชันสำหรับคำนวณอายุไก่ ณ วันปัจจุบัน (หน่วยเป็นวัน)
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
