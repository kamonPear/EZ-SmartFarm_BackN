package database

import (
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateImportFood records a new food import lot owned by userID and adds its volume
// onto userID's running Foodstock total for req.FoodType
func CreateImportFood(req *models.CreateImportFoodRequest, userID int) (*models.ImportFood, error) {
	lot := &models.ImportFood{
		UserID:       userID,
		FoodType:     req.FoodType,
		ImportVolume: req.ImportVolume,
		ImportDate:   time.Now(),
		ExpiryDate:   req.ExpiryDate,
	}

	if err := DB.Create(lot).Error; err != nil {
		log.Printf("Error creating import food lot: %v", err)
		return nil, err
	}

	if err := addToFoodstock(req.FoodType, userID, float64(req.ImportVolume)); err != nil {
		log.Printf("Error updating foodstock total after import: %v", err)
		return nil, err
	}

	log.Printf("✓ Import food lot #%d created (%s +%d kg) for user %d\n", lot.LotID, req.FoodType, req.ImportVolume, userID)
	return lot, nil
}

// addToFoodstock adds amount onto userID's foodstock row for foodType, creating it if it doesn't exist yet
func addToFoodstock(foodType string, userID int, amount float64) error {
	var foodstock models.Foodstock

	err := DB.Where("food_type = ? AND user_id = ?", foodType, userID).First(&foodstock).Error
	if err == nil {
		foodstock.QuantityCurrent += amount
		foodstock.DateUp = time.Now()
		return DB.Save(&foodstock).Error
	}

	if err != gorm.ErrRecordNotFound {
		return err
	}

	foodstock = models.Foodstock{
		UserID:          userID,
		FoodType:        foodType,
		QuantityCurrent: amount,
		DateUp:          time.Now(),
	}
	return DB.Create(&foodstock).Error
}

// GetImportFoodByID retrieves a single import food lot by ID, only if it's owned by userID
func GetImportFoodByID(lotID int, userID int) (*models.ImportFood, error) {
	var lot models.ImportFood
	if err := DB.Where("lot_id = ? AND user_id = ?", lotID, userID).First(&lot).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &lot, nil
}

// GetAllImportFood retrieves userID's full history of food import lots, most recent first
func GetAllImportFood(userID int) ([]models.ImportFood, error) {
	var lots []models.ImportFood
	if err := DB.Where("user_id = ?", userID).Order("lot_id desc").Find(&lots).Error; err != nil {
		log.Printf("Error fetching import food history: %v", err)
		return nil, err
	}
	return lots, nil
}

// DeleteAllImportFood wipes the caller's food import history (used when clearing the
// foodstock). Scoped to userID - "delete all" means all of *their* lots; a global wipe
// here would let any logged-in user destroy every other user's records.
func DeleteAllImportFood(userID int) error {
	if err := DB.Where("user_id = ?", userID).Delete(&models.ImportFood{}).Error; err != nil {
		log.Printf("Error deleting import food history: %v", err)
		return err
	}
	log.Println("✓ Import food history cleared")
	return nil
}
