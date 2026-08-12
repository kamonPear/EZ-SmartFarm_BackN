package models

import "time"

// Vaccine โครงสร้างสำหรับเก็บประวัติการฉีดวัคซีนจริงลงฐานข้อมูล
type Vaccine struct {
	VaccineID      int        `gorm:"primaryKey;column:vaccine_id" json:"vaccine_id"`
	
	CoopID         int        `gorm:"column:coop_id;type:int;index;not null" json:"coop_id"`
	NameCoop       string     `gorm:"column:name_coop;type:varchar(100);index" json:"name_coop"`
	
	Birthday       *time.Time `gorm:"column:birthday;type:date" json:"birthday"`
	// ✅ ตรงนี้คุณแก้มาถูกต้องแล้ว
	Name           string     `gorm:"column:name_vaccine;type:varchar(50);not null" json:"name"`
	RecordDate     time.Time  `gorm:"column:record_date;type:date;not null" json:"record_date"`
	Method         string     `gorm:"column:method;type:varchar(100);not null" json:"method"`
	RecommendedAge string     `gorm:"column:recommended_age;type:varchar(20);not null" json:"recommended_age"`
	Note           string     `gorm:"column:note;type:varchar(100)" json:"note"`
	
	// ถ้าไม่ต้องการให้ GORM ยุ่งกับ Constraint เลย ใส่ไว้อันนี้ถูกต้องแล้ว
	Coop           Coop       `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

func (Vaccine) TableName() string {
	return "vaccine"
}

// MedicineSchedule โครงสร้างตารางเงื่อนไขรอบการให้ยา/วัคซีนตามเกณฑ์อายุ
type MedicineSchedule struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Method      string `gorm:"column:method;type:varchar(100)" json:"method"`
	MinAgeDays  int    `gorm:"column:min_age_days;not null" json:"min_age_days"`
	MaxAgeDays  int    `gorm:"column:max_age_days;not null" json:"max_age_days"`
	Description string `gorm:"column:description;type:text" json:"description"`
}

func (MedicineSchedule) TableName() string {
	return "medicine_schedules"
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