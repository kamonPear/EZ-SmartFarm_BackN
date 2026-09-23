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

// FarmLayout is a singleton row (id=1) holding the chosen farm outline shape that
// coops are arranged within on the farm layout page.
type FarmLayout struct {
	ID    int    `gorm:"primaryKey;column:id;type:int" json:"id"`
	Shape string `gorm:"column:shape;type:varchar(20)" json:"shape"`
}

// TableName specifies the table name for FarmLayout model
func (FarmLayout) TableName() string {
	return "farm_layout"
}
