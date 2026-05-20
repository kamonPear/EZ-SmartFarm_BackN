package models

import "time"

// Egg represents egg collection records
type Egg struct {
	EggID          int       `gorm:"primaryKey;column:egg_id" json:"egg_id"`
	CoopID         int       `gorm:"column:coop_id;uniqueIndex:idx_coop_date" json:"coop_id"`
	DateCollectEgg time.Time `gorm:"column:date_collect_egg;uniqueIndex:idx_coop_date" json:"date_collect_egg"`
	NumberEgg      int       `gorm:"column:number_egg" json:"number_egg"`
	Note           string    `gorm:"column:note;type:text" json:"note"`

	// Relations
	Coop Coop `gorm:"foreignKey:CoopID;references:CoopID" json:"coop,omitempty"`
}

// TableName specifies the table name for Egg model
func (Egg) TableName() string {
	return "egg"
}
