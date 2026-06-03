package models

import "time"

// Coop represents the chicken coop/farm information
type Coop struct {
	CoopID           int       `gorm:"primaryKey;autoIncrement;column:coop_id;type:int" json:"coop_id"`
	DateAdoptAnimals time.Time `gorm:"column:date_adopt_animals" json:"date_adopt_animals"`
	Amount           int       `gorm:"column:amount" json:"amount"`
	Birthday         time.Time `gorm:"column:birthday;uniqueIndex" json:"birthday"`
	Note             string    `gorm:"column:note;type:text" json:"note"`

	// Relations (แก้โดยการลบ references:CoopID ออก)
	Devices  []Device  `gorm:"foreignKey:CoopID" json:"devices,omitempty"`
	Eggs     []Egg     `gorm:"foreignKey:CoopID" json:"eggs,omitempty"`
	Health   []Health  `gorm:"foreignKey:CoopID" json:"health,omitempty"`
	Vaccines []Vaccine `gorm:"foreignKey:CoopID" json:"vaccines,omitempty"`
}

// TableName specifies the table name for Coop model
func (Coop) TableName() string {
	return "coop"
}