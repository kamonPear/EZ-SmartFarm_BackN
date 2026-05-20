package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateCoop creates a new coop in the database
func CreateCoop(req *models.CreateCoopRequest) (*models.Coop, error) {
	coop := &models.Coop{
		DateAdoptAnimals: req.DateAdoptAnimals,
		Amount:           req.Amount,
		Birthday:         req.Birthday,
		Note:             req.Note,
	}

	if err := DB.Create(coop).Error; err != nil {
		log.Printf("Error creating coop: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Coop created with ID: %d\n", coop.CoopID)
	return coop, nil
}

// GetCoopByID retrieves a coop by ID
func GetCoopByID(coopID int) (*models.Coop, error) {
	var coop *models.Coop

	if err := DB.Preload("Devices").
		Preload("Eggs").
		Preload("Health").
		Preload("Vaccines").
		Where("coop_id = ?", coopID).
		First(&coop).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("coop not found")
		}
		log.Printf("Error fetching coop: %v", err)
		return nil, err
	}

	return coop, nil
}

// GetAllCoops retrieves all coops from the database
func GetAllCoops() ([]models.Coop, error) {
	var coops []models.Coop

	if err := DB.Preload("Devices").
		Preload("Eggs").
		Preload("Health").
		Preload("Vaccines").
		Find(&coops).Error; err != nil {
		log.Printf("Error fetching coops: %v", err)
		return nil, err
	}

	return coops, nil
}

// UpdateCoop updates an existing coop
func UpdateCoop(coopID int, req *models.UpdateCoopRequest) (*models.Coop, error) {
	coop, err := GetCoopByID(coopID)
	if err != nil {
		return nil, err
	}

	// Update only provided fields
	updates := map[string]interface{}{}

	if !req.DateAdoptAnimals.IsZero() {
		updates["date_adopt_animals"] = req.DateAdoptAnimals
	}
	if req.Amount > 0 {
		updates["amount"] = req.Amount
	}
	if !req.Birthday.IsZero() {
		updates["birthday"] = req.Birthday
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if len(updates) == 0 {
		return coop, nil
	}

	if err := DB.Model(coop).Updates(updates).Error; err != nil {
		log.Printf("Error updating coop: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Coop %d updated\n", coopID)
	return coop, nil
}

// DeleteCoop deletes a coop and all related records
func DeleteCoop(coopID int) error {
	coop, err := GetCoopByID(coopID)
	if err != nil {
		return err
	}

	// Use transaction to ensure all related records are deleted
	tx := DB.Begin()

	// Delete related records (cascading delete)
	if err := tx.Where("coop_id = ?", coopID).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting devices: %v", err)
		return err
	}

	if err := tx.Where("coop_id = ?", coopID).Delete(&models.Egg{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting eggs: %v", err)
		return err
	}

	if err := tx.Where("coop_id = ?", coopID).Delete(&models.Health{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting health records: %v", err)
		return err
	}

	if err := tx.Where("coop_id = ?", coopID).Delete(&models.Vaccine{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting vaccines: %v", err)
		return err
	}

	// Delete the coop itself
	if err := tx.Delete(coop).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting coop: %v", err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	fmt.Printf("✓ Coop %d and all related data deleted\n", coopID)
	return nil
}
