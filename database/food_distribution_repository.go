package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"
)

// CreateFoodDistributionBatch inserts one row per coop for a single deduction event.
// All rows share the same DistributedAt timestamp so they can be grouped back together.
func CreateFoodDistributionBatch(foodType string, items []models.FoodDistributionItem) ([]models.FoodDistribution, error) {
	now := time.Now()
	rows := make([]models.FoodDistribution, 0, len(items))
	for _, item := range items {
		rows = append(rows, models.FoodDistribution{
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

	fmt.Printf("✓ Recorded food distribution: %s across %d coops\n", foodType, len(rows))
	return rows, nil
}

// GetFoodDistributionHistory retrieves every recorded distribution, optionally filtered
// by food type, newest first.
func GetFoodDistributionHistory(foodType string) ([]models.FoodDistribution, error) {
	var rows []models.FoodDistribution
	query := DB.Preload("Coop").Order("distributed_at DESC")
	if foodType != "" {
		query = query.Where("food_type = ?", foodType)
	}
	if err := query.Find(&rows).Error; err != nil {
		log.Printf("Error fetching food distribution history: %v", err)
		return nil, err
	}
	return rows, nil
}
