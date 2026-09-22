package models

import "time"

// FoodDistribution records how many kg of a force-deducted food batch each coop
// actually received. One row per coop per deduction event, kept as permanent
// history (never overwritten) - see handlers.RecordFoodDistributionHandler.
type FoodDistribution struct {
	DistributionID int       `gorm:"primaryKey;autoIncrement;column:distribution_id;type:int" json:"distribution_id"`
	UserID         int       `gorm:"column:user_id;type:int;index" json:"user_id"`
	FoodType       string    `gorm:"column:food_type;type:varchar(20);not null;index" json:"food_type"`
	CoopID         int       `gorm:"column:coop_id;not null;type:int" json:"coop_id"`
	KgGiven        float64   `gorm:"column:kg_given;type:decimal(10,2);not null" json:"kg_given"`
	DistributedAt  time.Time `gorm:"column:distributed_at;not null" json:"distributed_at"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

// TableName specifies the table name for FoodDistribution model
func (FoodDistribution) TableName() string {
	return "food_distribution"
}
