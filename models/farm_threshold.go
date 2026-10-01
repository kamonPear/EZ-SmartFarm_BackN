package models

// FarmThreshold holds the target temperature/ammonia level the farm owner wants
// held steady, shared across every coop (not per-coop) - one row per user, created
// lazily with defaults on first read. Currently just a reference value shown on the
// dashboard; not yet wired to any automatic fan/light control.
type FarmThreshold struct {
	ID          int     `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID      int     `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_farm_threshold_user" json:"user_id"`
	Temperature float64 `gorm:"column:temperature;type:decimal(5,2);not null" json:"temperature"`
	Ammonia     float64 `gorm:"column:ammonia;type:decimal(5,2);not null" json:"ammonia"`
}

// TableName specifies the table name for FarmThreshold model
func (FarmThreshold) TableName() string {
	return "farm_threshold"
}

// Default target values for a user's first-ever farm threshold row.
const (
	DefaultFarmThresholdTemperature = 25.0
	DefaultFarmThresholdAmmonia     = 35.0
)
