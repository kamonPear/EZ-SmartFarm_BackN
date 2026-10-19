// =====================================================================================
// ไฟล์ models/vaccine_history.go - โครงสร้างตาราง vaccine_history (ประวัติการให้วัคซีนจริง)
// =====================================================================================
package models

import "time"

// VaccineHistory การให้วัคซีน/ยาจริง 1 ครั้งกับคอกหนึ่ง
// ทุกแถวคือเหตุการณ์ "ให้แล้ว" เสมอ coop_id / record_date จึงบังคับต้องมี (not null)
//   - NameCoop / Birthday : คัดลอกจากคอกตอนบันทึก (ประวัติแสดงข้อมูล ณ ตอนนั้น)
//   - Name               : ชื่อวัคซีน ใช้จับคู่กับประเภทในตาราง vaccine (คอลัมน์ name_vaccine)
//   - RecommendedAge     : อายุไก่ที่แนะนำให้ (เก็บเป็นข้อความ)
//
// ระบบดูว่า "คอกนี้ให้วัคซีนตัวนี้แล้วหรือยัง" จากการมีแถว (coop_id, name_vaccine) ในตารางนี้
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

// TableName บอก GORM ว่าใช้ตาราง "vaccine_history"
func (VaccineHistory) TableName() string {
	return "vaccine_history"
}
