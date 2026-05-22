package models

import "time"

// Foodstock represents food inventory/storage
type Foodstock struct {
	FoodID          int       `gorm:"primaryKey;autoIncrement;column:food_id;type:int" json:"food_id"`
	QuantityCurrent float64   `gorm:"column:quantity_current;type:decimal(10,2)" json:"quantity_current"`
	MinQuantity     float64   `gorm:"column:min_quantity;type:decimal(10,2);default:0" json:"min_quantity"`
	ImportDate      time.Time `gorm:"column:import_date" json:"import_date"`
	ExpiryDate      time.Time `gorm:"column:expiry_date" json:"expiry_date"`
	DateUp          time.Time `gorm:"column:date_up" json:"date_up"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for Foodstock model
func (Foodstock) TableName() string {
	return "foodstock"
}
