package models

import "time"

// Food type constants - every foodstock/import row must use one of these.
const (
	FoodTypeSmallPellet = "เม็ดเล็ก" // ไก่อายุ 0-6 สัปดาห์
	FoodTypeLargePellet = "เม็ดใหญ่" // ไก่อายุมากกว่า 6 สัปดาห์
)

// IsValidFoodType reports whether foodType is one of the known food types
func IsValidFoodType(foodType string) bool {
	return foodType == FoodTypeSmallPellet || foodType == FoodTypeLargePellet
}

// AgeThresholdWeeksSmallToLarge is the chicken age (in weeks) at which a coop switches
// from small pellet to large pellet feed.
const AgeThresholdWeeksSmallToLarge = 6

// FoodTypeForAgeWeeks returns which food type a coop should currently be fed, based on
// the age (in weeks) of the chickens raised there.
func FoodTypeForAgeWeeks(ageWeeks int) string {
	if ageWeeks <= AgeThresholdWeeksSmallToLarge {
		return FoodTypeSmallPellet
	}
	return FoodTypeLargePellet
}

// Foodstock represents the current running total of food in stock, kept up to date
// automatically. There is one row per FoodType (small pellet / large pellet).
type Foodstock struct {
	FoodID          int       `gorm:"primaryKey;autoIncrement;column:food_id;type:int" json:"food_id"`
	FoodType        string    `gorm:"column:food_type;type:varchar(20);uniqueIndex" json:"food_type"`
	QuantityCurrent float64   `gorm:"column:quantity_current;type:decimal(10,2);check:quantity_current >= 0" json:"quantity_current"`
	DateUp          time.Time `gorm:"column:date_up;type:date;not null" json:"date_up"`
}

// TableName specifies the table name for Foodstock model
func (Foodstock) TableName() string {
	return "foodstock"
}
