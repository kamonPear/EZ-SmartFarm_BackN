package models

// FarmThreshold holds the target temperature/ammonia level the farm owner wants
// held steady, shared across every coop (not per-coop) - one row per user, created
// only the first time that user sets a value (not auto-created with defaults on
// read, so an unconfigured user genuinely has no row and sees 0/0 - never another
// user's or a shared default's leftover value). Currently just a reference value
// shown on the dashboard; not yet wired to any automatic fan/light control.
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
