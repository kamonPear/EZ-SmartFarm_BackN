package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFarmThreshold retrieves userID's farm-wide target temperature/ammonia row.
// Returns a zero-valued (0/0) threshold WITHOUT writing a row to the database if the
// user hasn't set anything yet - "not configured" needs to stay a real, visible state
// per-user instead of every new user silently getting the same shared default values
// the moment their dashboard loads. A row only ever gets created by UpdateFarmThreshold,
// the first time the user actually sets a value themselves.
func GetFarmThreshold(userID int) (*models.FarmThreshold, error) {
	var threshold models.FarmThreshold

	err := DB.Where("user_id = ?", userID).First(&threshold).Error
	if err == nil {
		return &threshold, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("Error fetching farm threshold: %v", err)
		return nil, err
	}

	return &models.FarmThreshold{UserID: userID, Temperature: 0, Ammonia: 0}, nil
}

// UpdateFarmThreshold sets userID's target temperature/ammonia to whatever they submit,
// creating their row on first use (not a hardcoded default - the value they chose).
func UpdateFarmThreshold(req *models.UpdateFarmThresholdRequest, userID int) (*models.FarmThreshold, error) {
	var threshold models.FarmThreshold
	err := DB.Where("user_id = ?", userID).First(&threshold).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("Error fetching farm threshold: %v", err)
			return nil, err
		}
		threshold = models.FarmThreshold{UserID: userID}
		if err := DB.Create(&threshold).Error; err != nil {
			log.Printf("Error creating farm threshold: %v", err)
			return nil, err
		}
	}

	threshold.Temperature = req.Temperature
	threshold.Ammonia = req.Ammonia
	if err := DB.Save(&threshold).Error; err != nil {
		log.Printf("Error updating farm threshold: %v", err)
		return nil, err
	}

	log.Printf("✓ Farm threshold set to %.1f°C / %.1f ppm for user %d\n", req.Temperature, req.Ammonia, userID)
	return &threshold, nil
}
