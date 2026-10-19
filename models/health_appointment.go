// =====================================================================================
// ไฟล์ models/health_appointment.go - โครงสร้างตาราง health_appointment (นัดตรวจสุขภาพที่ผู้ใช้ตั้งเอง)
// =====================================================================================
package models

import "time"

// HealthAppointment นัดตรวจสุขภาพคอกที่ผู้ใช้เลือกวันเอง (ฟีเจอร์ "นัดตรวจสุขภาพเอง")
// แยกจากนัดที่ระบบคำนวณจากวัคซีน เช็กเจ้าของผ่านคอก (เหมือน egg/health)
// CreatedAt มี autoCreateTime = GORM ใส่เวลาที่สร้างให้อัตโนมัติ
type HealthAppointment struct {
	AppointmentID   int       `gorm:"primaryKey;autoIncrement;column:appointment_id;type:int" json:"appointment_id"`
	CoopID          int       `gorm:"column:coop_id;type:int;size:32;index;not null" json:"coop_id"`
	AppointmentDate time.Time `gorm:"column:appointment_date;type:date;index;not null" json:"appointment_date"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

// TableName บอก GORM ว่าใช้ตาราง "health_appointment"
func (HealthAppointment) TableName() string {
	return "health_appointment"
}
