package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFoodAgeConfig returns userID's food/age ranges, or the default rule (small 0-14,
// large 14+) WITHOUT writing a row if the owner hasn't configured anything yet.
func GetFoodAgeConfig(userID int) (*models.FoodAgeConfig, error) {
	var cfg models.FoodAgeConfig
	err := DB.Where("user_id = ?", userID).First(&cfg).Error
	if err == nil {
		cfg.Configured = true
		return &cfg, nil
	}
	if err != gorm.ErrRecordNotFound {
		log.Printf("Error fetching food age config: %v", err)
		return nil, err
	}
	def := models.DefaultFoodAgeConfig(userID)
	return &def, nil
}

// UpdateFoodAgeConfig saves userID's ranges, creating the row on first use.
func UpdateFoodAgeConfig(req *models.UpdateFoodAgeConfigRequest, userID int) (*models.FoodAgeConfig, error) {
	var cfg models.FoodAgeConfig
	err := DB.Where("user_id = ?", userID).First(&cfg).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("Error fetching food age config: %v", err)
			return nil, err
		}
		cfg = models.FoodAgeConfig{UserID: userID}
	}
	cfg.SmallMinWeeks = req.SmallMinWeeks
	cfg.SmallMaxWeeks = req.SmallMaxWeeks
	cfg.LargeMinWeeks = req.LargeMinWeeks
	cfg.LargeMaxWeeks = req.LargeMaxWeeks
	if err := DB.Save(&cfg).Error; err != nil {
		log.Printf("Error saving food age config: %v", err)
		return nil, err
	}
	cfg.Configured = true
	return &cfg, nil
}
