package models

import "time"

// HealthAppointment is a health-check appointment the farm owner books manually for
// a coop on a specific future date (the "นัดตรวจสุขภาพเอง" flow) - independent of the
// automatically-computed vaccine-based appointment (1 day before a due vaccine).
// Scoped through its coop the same way Health/Egg/Vaccine already are.
type HealthAppointment struct {
	AppointmentID   int       `gorm:"primaryKey;autoIncrement;column:appointment_id;type:int" json:"appointment_id"`
	CoopID          int       `gorm:"column:coop_id;type:int;size:32;index;not null" json:"coop_id"`
	AppointmentDate time.Time `gorm:"column:appointment_date;type:date;index;not null" json:"appointment_date"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

func (HealthAppointment) TableName() string {
	return "health_appointment"
}
