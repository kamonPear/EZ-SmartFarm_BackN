// =====================================================================================
// ไฟล์ models/health.go - โครงสร้างตาราง health (ผลตรวจสุขภาพไก่)
// =====================================================================================
package models

import "time"

// Health ผลตรวจสุขภาพ 1 ครั้งของคอก
//  - Healthy    -> คอลัมน์ number_healthy     : จำนวนไก่สุขภาพดี
//  - PoorHealth -> คอลัมน์ number_poor_health : จำนวนไก่ป่วย
//  - RecordDate -> คอลัมน์ date               : วันที่ตรวจ
// สังเกตว่าชื่อ field ใน Go / JSON ไม่ตรงกับชื่อคอลัมน์ (ดู gorm:"column:...")
// ตอนอัปเดตด้วย map จึงต้องใช้ชื่อคอลัมน์จริง (ดู UpdateHealth)
// 1 คอก 1 วันมีได้ 1 แถว (unique uq_health_coop_date)
type Health struct {
	HealthID   int       `gorm:"primaryKey;column:health_id;type:int" json:"health_id"`
	CoopID     int       `gorm:"column:coop_id;index;not null;type:int;size:32" json:"coop_id"`
	Healthy    int       `gorm:"column:number_healthy;default:0" json:"healthy"`
	PoorHealth int       `gorm:"column:number_poor_health;default:0" json:"poor_health"`
	Note       string    `gorm:"column:note;type:text" json:"note"`
	RecordDate time.Time `gorm:"column:date;index" json:"record_date"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

// TableName บอก GORM ว่าใช้ตาราง "health"
func (Health) TableName() string {
	return "health"
}
