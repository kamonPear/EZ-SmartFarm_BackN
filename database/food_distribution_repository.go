package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"
)

// CreateFoodDistributionBatch inserts one row per coop for a single deduction event,
// owned by userID. All rows share the same DistributedAt timestamp so they can be
// grouped back together.
func CreateFoodDistributionBatch(foodType string, items []models.FoodDistributionItem, userID int) ([]models.FoodDistribution, error) {
	now := time.Now()
	rows := make([]models.FoodDistribution, 0, len(items))
	for _, item := range items {
		rows = append(rows, models.FoodDistribution{
			UserID:        userID,
			FoodType:      foodType,
			CoopID:        item.CoopID,
			KgGiven:       item.KgGiven,
			DistributedAt: now,
		})
	}

	if err := DB.Create(&rows).Error; err != nil {
		log.Printf("Error creating food distribution batch: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Recorded food distribution: %s across %d coops (user %d)\n", foodType, len(rows), userID)
	return rows, nil
}

// GetFoodDistributionHistory retrieves every distribution recorded by userID,
// optionally filtered by food type, newest first.
func GetFoodDistributionHistory(foodType string, userID int) ([]models.FoodDistribution, error) {
	var rows []models.FoodDistribution
	query := DB.Preload("Coop").Where("user_id = ?", userID).Order("distributed_at DESC")
	if foodType != "" {
		query = query.Where("food_type = ?", foodType)
	}
	if err := query.Find(&rows).Error; err != nil {
		log.Printf("Error fetching food distribution history: %v", err)
		return nil, err
	}
	return rows, nil
}

// DeleteAllFoodDistribution wipes the entire food distribution history
func DeleteAllFoodDistribution(userID int) error {
	// Scoped to the caller - "delete all" means all of *their* history. A global
	// wipe here would let any logged-in user destroy every other user's records.
	if err := DB.Where("user_id = ?", userID).Delete(&models.FoodDistribution{}).Error; err != nil {
		log.Printf("Error deleting food distribution history: %v", err)
		return err
	}
	log.Println("✓ Food distribution history cleared")
	return nil
}
