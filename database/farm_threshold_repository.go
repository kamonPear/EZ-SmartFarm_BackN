package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFarmThreshold retrieves userID's farm-wide target temperature/ammonia row,
// creating it with sensible defaults on first read if it doesn't exist yet.
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

	threshold = models.FarmThreshold{
		UserID:      userID,
		Temperature: models.DefaultFarmThresholdTemperature,
		Ammonia:     models.DefaultFarmThresholdAmmonia,
	}
	if err := DB.Create(&threshold).Error; err != nil {
		log.Printf("Error creating default farm threshold: %v", err)
		return nil, err
	}

	log.Printf("✓ Created default farm threshold for user %d\n", userID)
	return &threshold, nil
}

// UpdateFarmThreshold sets userID's target temperature/ammonia, creating their row
// first if it doesn't exist yet.
func UpdateFarmThreshold(req *models.UpdateFarmThresholdRequest, userID int) (*models.FarmThreshold, error) {
	threshold, err := GetFarmThreshold(userID)
	if err != nil {
		return nil, err
	}

	threshold.Temperature = req.Temperature
	threshold.Ammonia = req.Ammonia
	if err := DB.Save(threshold).Error; err != nil {
		log.Printf("Error updating farm threshold: %v", err)
		return nil, err
	}

	log.Printf("✓ Farm threshold set to %.1f°C / %.1f ppm for user %d\n", req.Temperature, req.Ammonia, userID)
	return threshold, nil
}
