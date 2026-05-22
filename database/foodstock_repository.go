package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateFoodstock creates a new foodstock in the database 
func CreateFoodstock(req *models.CreateFoodstockRequest) (*models.Foodstock, error) {
	foodstock := &models.Foodstock{
		QuantityCurrent: req.QuantityCurrent,
		MinQuantity:     req.MinQuantity,
		ImportDate:      req.ImportDate,
		ExpiryDate:      req.ExpiryDate,
		DateUp:          req.DateUp,
	}

	if err := DB.Create(foodstock).Error; err != nil {
		log.Printf("Error creating foodstock: %v", err)
		return nil, err
	}

	log.Printf("✓ Foodstock created with ID: %d\n", foodstock.FoodID)
	return foodstock, nil
}

// GetFoodstockByID retrieves a foodstock by ID
func GetFoodstockByID(foodID int) (*models.Foodstock, error) {
	var foodstock *models.Foodstock

	if err := DB.Where("food_id = ?", foodID).
		First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("foodstock not found")
		}
		log.Printf("Error fetching foodstock: %v", err)
		return nil, err
	}

	return foodstock, nil
}

// GetAllFoodstocks retrieves all foodstocks from the database
func GetAllFoodstocks() ([]models.Foodstock, error) {
	var foodstocks []models.Foodstock

	if err := DB.Find(&foodstocks).Error; err != nil {
		log.Printf("Error fetching foodstocks: %v", err)
		return nil, err
	}

	return foodstocks, nil
}

// UpdateFoodstock updates an existing foodstock
func UpdateFoodstock(foodID int, req *models.UpdateFoodstockRequest) (*models.Foodstock, error) {
	foodstock, err := GetFoodstockByID(foodID)
	if err != nil {
		return nil, err
	}

	// Update only provided fields
	updates := map[string]interface{}{}

	if req.QuantityCurrent >= 0 {
		updates["quantity_current"] = req.QuantityCurrent
	}
	if req.MinQuantity >= 0 {
		updates["min_quantity"] = req.MinQuantity
	}
	if !req.ImportDate.IsZero() {
		updates["import_date"] = req.ImportDate
	}
	if !req.ExpiryDate.IsZero() {
		updates["expiry_date"] = req.ExpiryDate
	}
	if !req.DateUp.IsZero() {
		updates["date_up"] = req.DateUp
	}

	if len(updates) == 0 {
		return foodstock, nil
	}

	if err := DB.Model(foodstock).Updates(updates).Error; err != nil {
		log.Printf("Error updating foodstock: %v", err)
		return nil, err
	}

	log.Printf("✓ Foodstock %d updated\n", foodID)
	return foodstock, nil
}

// DeleteFoodstock deletes a foodstock from the database
func DeleteFoodstock(foodID int) error {
	foodstock, err := GetFoodstockByID(foodID)
	if err != nil {
		return err
	}

	if err := DB.Delete(foodstock).Error; err != nil {
		log.Printf("Error deleting foodstock: %v", err)
		return err
	}

	log.Printf("✓ Foodstock %d deleted\n", foodID)
	return nil
}
