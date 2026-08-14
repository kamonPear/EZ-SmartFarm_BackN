package models

import "time"

// ImportFood represents a single food import lot (history of every food import)
type ImportFood struct {
	LotID        int       `gorm:"primaryKey;autoIncrement;column:lot_id;type:int" json:"lot_id"`
	ImportVolume int       `gorm:"column:import_volume;type:int;not null" json:"import_volume"`
	ImportDate   time.Time `gorm:"column:import_date;type:datetime;not null" json:"import_date"`
	ExpiryDate   time.Time `gorm:"column:expiry_date;type:date;not null" json:"expiry_date"`
}

// TableName specifies the table name for ImportFood model
func (ImportFood) TableName() string {
	return "importfood"
}
