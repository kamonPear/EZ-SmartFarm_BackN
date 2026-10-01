package models

import (
	"time"

	"gorm.io/gorm"
)

// Vaccine เก็บทั้ง "ประเภทยา/วัคซีน" (เกณฑ์ช่วงอายุ ไม่ผูกคอกไหน) และ "ประวัติการ
// ให้จริง" (ผูกกับคอกใดคอกหนึ่ง) ไว้ในตารางเดียวกัน แยกกันด้วย CoopID:
//   - CoopID == nil  -> แถว "ประเภท" (template) ใช้ MinAgeDays/MaxAgeDays กำหนด
//     ช่วงอายุที่ควรให้ ไม่ผูกกับผู้ใช้คนไหน (เหมือน medicine_schedules เดิม)
//   - CoopID != nil  -> แถว "ประวัติให้จริง" ของคอกนั้น ใช้ RecordDate/RecommendedAge
//     (อายุไก่ ณ วันที่ให้) ไม่ใช้ MinAgeDays/MaxAgeDays
// (เดิมแยกเป็น 2 ตาราง คือ medicine_schedules กับ vaccine - รวมเป็นตารางเดียวตามที่
// ผู้ใช้ขอ เพื่อไม่ให้สับสนว่า "เพิ่มวัคซีน" แล้วทำไมไม่เห็นในตาราง vaccine)
type Vaccine struct {
	VaccineID int `gorm:"primaryKey;column:vaccine_id" json:"vaccine_id"`

	// nil = แถว "ประเภท" ยังไม่ได้ผูกกับคอกไหน - ต้องเป็น pointer ไม่งั้นบังคับใส่
	// ค่าเสมอ (รวมถึง 0 ซึ่งไม่ใช่ coop จริง) ตอนสร้างแถวประเภทไม่ได้
	CoopID   *int       `gorm:"column:coop_id;type:int;size:32;index" json:"coop_id"`
	NameCoop string     `gorm:"column:name_coop;type:varchar(100);index" json:"name_coop"`
	Birthday *time.Time `gorm:"column:birthday;type:date" json:"birthday"`

	Name string `gorm:"column:name_vaccine;type:varchar(50);not null" json:"name"`
	// LegacyName เก็บค่าเดียวกับ Name เพื่อเติมคอลัมน์ "name" (ของเดิม, NOT NULL, ไม่มี default)
	// ที่ยังหลงเหลืออยู่ในตาราง - ถ้าไม่เซ็ตตัวนี้ INSERT ผ่าน struct ปกติจะโดน MySQL error 1364
	LegacyName string `gorm:"column:name;type:varchar(50);not null" json:"-"`
	Method     string `gorm:"column:method;type:varchar(100);not null" json:"method"`
	Note       string `gorm:"column:note;type:varchar(100)" json:"note"`

	// ใช้เฉพาะแถว "ประวัติให้จริง" (CoopID != nil)
	RecordDate     *time.Time `gorm:"column:record_date;type:date" json:"record_date,omitempty"`
	RecommendedAge string     `gorm:"column:recommended_age;type:varchar(20)" json:"recommended_age,omitempty"`

	// ใช้เฉพาะแถว "ประเภท" (CoopID == nil)
	MinAgeDays *int `gorm:"column:min_age_days;type:int" json:"min_age_days,omitempty"`
	MaxAgeDays *int `gorm:"column:max_age_days;type:int" json:"max_age_days,omitempty"`

	// ถ้าไม่ต้องการให้ GORM ยุ่งกับ Constraint เลย ใส่ไว้อันนี้ถูกต้องแล้ว
	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

func (Vaccine) TableName() string {
	return "vaccine"
}

// BeforeSave keeps the legacy "name" column in sync with Name so every GORM
// write path (Create/Save on the struct) satisfies the NOT NULL constraint
// without callers having to remember to set LegacyName themselves.
func (v *Vaccine) BeforeSave(tx *gorm.DB) error {
	v.LegacyName = v.Name
	return nil
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
