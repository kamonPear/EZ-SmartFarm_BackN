package models

import "time"

// Vaccine represents vaccination records
type Vaccine struct {
	VaccineID      int        `gorm:"primaryKey;column:vaccine_id;type:int" json:"vaccine_id"`
	CoopID         int        `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
	Birthday       *time.Time `gorm:"column:birthday;type:date" json:"birthday"`
	Name           string     `gorm:"column:name;type:varchar(50);not null" json:"name"`
	RecordDate     time.Time  `gorm:"column:record_date;type:date;not null" json:"record_date"`
	Method         string     `gorm:"column:method;type:varchar(100);not null" json:"method"`
	RecommendedAge string     `gorm:"column:recommended_age;type:varchar(20);not null" json:"recommended_age"`
	Note           string     `gorm:"column:note;type:varchar(100)" json:"note"`

	// Relations
	Coop Coop `gorm:"foreignKey:CoopID;references:CoopID" json:"coop,omitempty"`
}

// TableName specifies the table name for Vaccine model

func (Vaccine) TableName() string {
	return "vaccine"
}


