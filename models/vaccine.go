package models

import "time"

// Vaccine represents vaccination records
type Vaccine struct {
	VaccineID      int       `gorm:"primaryKey;column:vaccine_id" json:"vaccine_id"`
	CoopID         int       `gorm:"column:coop_id;index" json:"coop_id"`
	Birthday       time.Time `gorm:"column:birthday" json:"birthday"`
	Name           string    `gorm:"column:name;type:varchar(50)" json:"name"`
	RecordDate     time.Time `gorm:"column:record_date" json:"record_date"`
	Method         string    `gorm:"column:method;type:varchar(100)" json:"method"`
	RecommendedAge string    `gorm:"column:recommended_age;type:varchar(20)" json:"recommended_age"`
	Note           string    `gorm:"column:note;type:varchar(100)" json:"note"`

	// Relations
	Coop Coop `gorm:"foreignKey:CoopID;references:CoopID" json:"coop,omitempty"`
}

// TableName specifies the table name for Vaccine model
func (Vaccine) TableName() string {
	return "vaccine"
}
