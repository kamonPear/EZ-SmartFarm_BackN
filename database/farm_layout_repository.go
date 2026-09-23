package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFarmLayout retrieves userID's farm layout row, creating it with the default
// shape (circle) on first read if it doesn't exist yet.
func GetFarmLayout(userID int) (*models.FarmLayout, error) {
	var layout models.FarmLayout

	err := DB.Where("user_id = ?", userID).First(&layout).Error
	if err == nil {
		return &layout, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("Error fetching farm layout: %v", err)
		return nil, err
	}

	layout = models.FarmLayout{UserID: userID, Shape: models.FarmShapeCircle}
	if err := DB.Create(&layout).Error; err != nil {
		log.Printf("Error creating default farm layout: %v", err)
		return nil, err
	}

	log.Printf("✓ Created default farm layout (circle) for user %d\n", userID)
	return &layout, nil
}

// UpdateFarmLayoutShape sets userID's chosen outline shape, creating their row first
// if it doesn't exist yet.
func UpdateFarmLayoutShape(shape string, userID int) (*models.FarmLayout, error) {
	layout, err := GetFarmLayout(userID)
	if err != nil {
		return nil, err
	}

	layout.Shape = shape
	if err := DB.Save(layout).Error; err != nil {
		log.Printf("Error updating farm layout shape: %v", err)
		return nil, err
	}

	log.Printf("✓ Farm layout shape set to %s for user %d\n", shape, userID)
	return layout, nil
}
