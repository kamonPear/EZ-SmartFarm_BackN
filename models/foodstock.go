package models

import "time"

// Food type constants - every foodstock/import row must use one of these.
const (
	FoodTypeSmallPellet = "เม็ดเล็ก" // ไก่อายุ 0-14 สัปดาห์
	FoodTypeLargePellet = "เม็ดใหญ่" // ไก่อายุตั้งแต่ 14 สัปดาห์ขึ้นไป
)

// IsValidFoodType reports whether foodType is one of the known food types
func IsValidFoodType(foodType string) bool {
	return foodType == FoodTypeSmallPellet || foodType == FoodTypeLargePellet
}

// Foodstock represents the current running total of food in stock, kept up to date
// automatically. There is one row per FoodType (small pellet / large pellet).
type Foodstock struct {
	FoodID          int       `gorm:"primaryKey;autoIncrement;column:food_id;type:int" json:"food_id"`
	UserID          int       `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_foodstock_type_user" json:"user_id"`
	FoodType        string    `gorm:"column:food_type;type:varchar(20);uniqueIndex:uq_foodstock_type_user" json:"food_type"`
	QuantityCurrent float64   `gorm:"column:quantity_current;type:decimal(10,2);check:quantity_current >= 0" json:"quantity_current"`
	DateUp          time.Time `gorm:"column:date_up;type:date;not null" json:"date_up"`
}

// TableName specifies the table name for Foodstock model
func (Foodstock) TableName() string {
	return "foodstock"
}
