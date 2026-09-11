package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFarmLayout retrieves the singleton farm layout row, creating it with the default
// shape (circle) on first read if it doesn't exist yet.
func GetFarmLayout() (*models.FarmLayout, error) {
	var layout models.FarmLayout

	err := DB.Where("id = ?", 1).First(&layout).Error
	if err == nil {
		return &layout, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("Error fetching farm layout: %v", err)
		return nil, err
	}

	layout = models.FarmLayout{ID: 1, Shape: models.FarmShapeCircle}
	if err := DB.Create(&layout).Error; err != nil {
		log.Printf("Error creating default farm layout: %v", err)
		return nil, err
	}

	log.Println("✓ Created default farm layout (circle)")
	return &layout, nil
}

// UpdateFarmLayoutShape sets the farm's chosen outline shape, creating the singleton
// row first if it doesn't exist yet.
func UpdateFarmLayoutShape(shape string) (*models.FarmLayout, error) {
	layout, err := GetFarmLayout()
	if err != nil {
		return nil, err
	}

	layout.Shape = shape
	if err := DB.Save(layout).Error; err != nil {
		log.Printf("Error updating farm layout shape: %v", err)
		return nil, err
	}

	log.Printf("✓ Farm layout shape set to %s\n", shape)
	return layout, nil
}
