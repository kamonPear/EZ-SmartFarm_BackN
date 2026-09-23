package models

// Farm layout shape constants
const (
	FarmShapeCircle   = "circle"
	FarmShapeTriangle = "triangle"
	FarmShapeSquare   = "square"
)

// IsValidFarmShape reports whether shape is one of the known farm layout shapes
func IsValidFarmShape(shape string) bool {
	return shape == FarmShapeCircle || shape == FarmShapeTriangle || shape == FarmShapeSquare
}

// FarmLayout holds the chosen farm outline shape that a user's coops are arranged
// within on the farm layout page. Used to be a singleton row (id=1) shared by
// everyone; now each user gets their own row (one per user_id), created lazily on
// first read/write, per the per-user data ownership model.
type FarmLayout struct {
	ID     int    `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID int    `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_farm_layout_user" json:"user_id"`
	Shape  string `gorm:"column:shape;type:varchar(20)" json:"shape"`
}

// TableName specifies the table name for FarmLayout model
func (FarmLayout) TableName() string {
	return "farm_layout"
}
