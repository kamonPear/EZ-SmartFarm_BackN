package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFarmLayout retrieves userID's farm layout row. Returns a row with an empty
// Shape WITHOUT writing anything to the database if the user hasn't chosen a shape
// yet - a row is only ever created by UpdateFarmLayoutShape, the first time the user
// actually picks one (see the same reasoning on GetFarmThreshold in
// farm_threshold_repository.go - every user's dashboard load used to silently create
// a "circle" row for them even if they'd never opened the layout page).
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

	return &models.FarmLayout{UserID: userID, Shape: ""}, nil
}

// UpdateFarmLayoutShape sets userID's chosen outline shape to whatever they submit,
// creating their row on first use.
func UpdateFarmLayoutShape(shape string, userID int) (*models.FarmLayout, error) {
	var layout models.FarmLayout
	err := DB.Where("user_id = ?", userID).First(&layout).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("Error fetching farm layout: %v", err)
			return nil, err
		}
		layout = models.FarmLayout{UserID: userID}
		if err := DB.Create(&layout).Error; err != nil {
			log.Printf("Error creating farm layout: %v", err)
			return nil, err
		}
	}

	layout.Shape = shape
	if err := DB.Save(&layout).Error; err != nil {
		log.Printf("Error updating farm layout shape: %v", err)
		return nil, err
	}

	log.Printf("✓ Farm layout shape set to %s for user %d\n", shape, userID)
	return &layout, nil
}
