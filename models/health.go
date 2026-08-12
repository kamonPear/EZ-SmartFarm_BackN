package models

import "time"

// Health represents chicken health records
type Health struct {
	HealthID   int       `gorm:"primaryKey;column:health_id;type:int" json:"health_id"`
	CoopID     int       `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
	NameCoop   string    `gorm:"column:name_coop;type:varchar(100);index" json:"name_coop"`
	Healthy    int       `gorm:"column:number_healthy;default:0" json:"healthy"`
	PoorHealth int       `gorm:"column:number_poor_health;default:0" json:"poor_health"`
	Note       string    `gorm:"column:note;type:text" json:"note"`
	RecordDate time.Time `gorm:"column:date;index" json:"record_date"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

// TableName specifies the table name for Health model
func (Health) TableName() string {
	return "health"
}