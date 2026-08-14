package database

import (
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateImportFood records a new food import lot and adds its volume onto the running Foodstock total (food_id = 1)
func CreateImportFood(req *models.CreateImportFoodRequest) (*models.ImportFood, error) {
	lot := &models.ImportFood{
		ImportVolume: req.ImportVolume,
		ImportDate:   time.Now(),
		ExpiryDate:   req.ExpiryDate,
	}

	if err := DB.Create(lot).Error; err != nil {
		log.Printf("Error creating import food lot: %v", err)
		return nil, err
	}

	if err := addToFoodstock(float64(req.ImportVolume)); err != nil {
		log.Printf("Error updating foodstock total after import: %v", err)
		return nil, err
	}

	log.Printf("✓ Import food lot #%d created (+%d kg)\n", lot.LotID, req.ImportVolume)
	return lot, nil
}

// addToFoodstock adds amount onto the singleton foodstock row (food_id = 1), creating it if it doesn't exist yet
func addToFoodstock(amount float64) error {
	var foodstock models.Foodstock

	err := DB.Where("food_id = ?", 1).First(&foodstock).Error
	if err == nil {
		foodstock.QuantityCurrent += amount
		foodstock.DateUp = time.Now()
		return DB.Save(&foodstock).Error
	}

	if err != gorm.ErrRecordNotFound {
		return err
	}

	foodstock = models.Foodstock{
		FoodID:          1,
		QuantityCurrent: amount,
		DateUp:          time.Now(),
	}
	return DB.Create(&foodstock).Error
}

// GetImportFoodByID retrieves a single import food lot by ID
func GetImportFoodByID(lotID int) (*models.ImportFood, error) {
	var lot models.ImportFood
	if err := DB.Where("lot_id = ?", lotID).First(&lot).Error; err != nil {
		return nil, err
	}
	return &lot, nil
}

// GetAllImportFood retrieves the full history of food import lots, most recent first
func GetAllImportFood() ([]models.ImportFood, error) {
	var lots []models.ImportFood
	if err := DB.Order("lot_id desc").Find(&lots).Error; err != nil {
		log.Printf("Error fetching import food history: %v", err)
		return nil, err
	}
	return lots, nil
}

// DeleteAllImportFood wipes the entire food import history (used when clearing the foodstock)
func DeleteAllImportFood() error {
	if err := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.ImportFood{}).Error; err != nil {
		log.Printf("Error deleting import food history: %v", err)
		return err
	}
	log.Println("✓ Import food history cleared")
	return nil
}
