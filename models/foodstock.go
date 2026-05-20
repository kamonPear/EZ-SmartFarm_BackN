package models

import "time"

// Foodstock represents food inventory/storage
type Foodstock struct {
	FoodID          int       `gorm:"primaryKey;column:food_id" json:"food_id"`
	QuantityCurrent float64   `gorm:"column:quantity_current;type:decimal(10,2)" json:"quantity_current"`
	ImportDate      time.Time `gorm:"column:import_date" json:"import_date"`
	ExpiryDate      time.Time `gorm:"column:expiry_date" json:"expiry_date"`
	DateUp          time.Time `gorm:"column:date_up;autoUpdateTime:milli" json:"date_up"`
}

// TableName specifies the table name for Foodstock model
func (Foodstock) TableName() string {
	return "foodstock"
}
