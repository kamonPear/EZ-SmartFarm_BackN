package models

import "time"

// Health represents chicken health records
type Health struct {
	HealthID   int       `gorm:"primaryKey;column:health_id;type:int" json:"health_id"`
	CoopID     int       `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
	Healthy    int       `gorm:"column:healthy;default:0" json:"healthy"`
	PoorHealth int       `gorm:"column:poor_health;default:0" json:"poor_health"`
	Note       string    `gorm:"column:note;type:text" json:"note"`
	RecordDate time.Time `gorm:"column:record_date;index" json:"record_date"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

// TableName specifies the table name for Health model
func (Health) TableName() string {
	return "health"
}
