package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"
)

// GetFoodAgeRules returns userID's whole feeding table, ordered by starting age.
// An owner who hasn't filled anything in simply has no rows (empty slice).
func GetFoodAgeRules(userID int) (models.FoodRules, error) {
	rules := models.FoodRules{}
	if err := DB.Where("user_id = ?", userID).Order("min_weeks ASC").Find(&rules).Error; err != nil {
		log.Printf("Error fetching food age rules: %v", err)
		return nil, err
	}
	return rules, nil
}

// ReplaceFoodAgeRules swaps userID's whole feeding table for the submitted rows in one
// transaction (the web popup edits the table as a list, so saving = replace all).
func ReplaceFoodAgeRules(userID int, rows []models.FoodAgeRuleInput) (models.FoodRules, error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.FoodAgeRule{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error clearing food age rules: %v", err)
		return nil, err
	}
	for _, in := range rows {
		rule := models.FoodAgeRule{
			UserID:       userID,
			FoodType:     in.FoodType,
			MinWeeks:     in.MinWeeks,
			MaxWeeks:     in.MaxWeeks,
			GramsPerBird: in.GramsPerBird,
		}
		if err := tx.Create(&rule).Error; err != nil {
			tx.Rollback()
			log.Printf("Error saving food age rule: %v", err)
			return nil, err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return GetFoodAgeRules(userID)
}
