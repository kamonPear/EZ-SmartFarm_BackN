package models

import "time"

// Foodstock represents the current running total of food in stock (single row, kept up to date automatically)
type Foodstock struct {
	FoodID          int       `gorm:"primaryKey;autoIncrement;column:food_id;type:int" json:"food_id"`
	QuantityCurrent float64   `gorm:"column:quantity_current;type:decimal(10,2);check:quantity_current >= 0" json:"quantity_current"`
	DateUp          time.Time `gorm:"column:date_up;type:date;not null" json:"date_up"`
}

// TableName specifies the table name for Foodstock model
func (Foodstock) TableName() string {
	return "foodstock"
}
