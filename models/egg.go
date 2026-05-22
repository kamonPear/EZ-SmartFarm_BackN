package models

import "time"

// Egg represents egg collection records
type Egg struct {
	EggID          int       `gorm:"primaryKey;autoIncrement;column:egg_id;type:int" json:"egg_id"`
	CoopID         int       `gorm:"column:coop_id;not null;type:int" json:"coop_id"`
	DateCollectEgg time.Time `gorm:"column:date_collect_egg" json:"date_collect_egg"`
	NumberEgg      int       `gorm:"column:number_egg" json:"number_egg"`
	Note           string    `gorm:"column:note;type:text" json:"note"`

	// Relations
	Coop Coop `gorm:"foreignKey:CoopID;references:CoopID" json:"coop,omitempty"`
}

// TableName specifies the table name for Egg model
func (Egg) TableName() string {
	return "egg"
}
