package models

import "time"

// Coop represents the chicken coop/farm information
type Coop struct {
	CoopID           int       `gorm:"primaryKey;autoIncrement;column:coop_id;type:int" json:"coop_id"`
	DateAdoptAnimals time.Time `gorm:"column:date_adopt_animals" json:"date_adopt_animals"`
	Amount           int       `gorm:"column:amount" json:"amount"`
	Birthday         time.Time `gorm:"column:birthday;uniqueIndex" json:"birthday"`
	Note             string    `gorm:"column:note;type:text" json:"note"`

	// Relations
	Devices  []Device  `gorm:"foreignKey:CoopID;references:CoopID" json:"devices,omitempty"`
	Eggs     []Egg     `gorm:"foreignKey:CoopID;references:CoopID" json:"eggs,omitempty"`
	Health   []Health  `gorm:"foreignKey:CoopID;references:CoopID" json:"health,omitempty"`
	Vaccines []Vaccine `gorm:"foreignKey:CoopID;references:CoopID" json:"vaccines,omitempty"`
}

// TableName specifies the table name for Coop model
func (Coop) TableName() string {
	return "coop"
}
