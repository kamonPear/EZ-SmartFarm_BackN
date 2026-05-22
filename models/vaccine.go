package models

// Vaccine represents vaccination records
type Vaccine struct {
	VaccineID      int    `gorm:"primaryKey;column:vaccine_id;type:int" json:"vaccine_id"`
	CoopID         int    `gorm:"column:coop_id;index;not null;type:int" json:"coop_id"`
	Method         string `gorm:"column:method;type:varchar(100)" json:"method"`
	RecommendedAge string `gorm:"column:recommended_age;type:varchar(20)" json:"recommended_age"`
	Note           string `gorm:"column:note;type:varchar(100)" json:"note"`

	// Relations
	Coop Coop `gorm:"foreignKey:CoopID;references:CoopID" json:"coop,omitempty"`
}

// TableName specifies the table name for Vaccine model

func (Vaccine) TableName() string {
	return "vaccine"
}
