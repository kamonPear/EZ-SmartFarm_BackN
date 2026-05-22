package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateEgg creates a new egg record in the database
func CreateEgg(req *models.CreateEggRequest) (*models.Egg, error) {
	// Verify that the coop exists
	if _, err := GetCoopByID(req.CoopID); err != nil {
		return nil, fmt.Errorf("coop not found: %v", err)
	}

	egg := &models.Egg{
		CoopID:         req.CoopID,
		DateCollectEgg: req.DateCollectEgg,
		NumberEgg:      req.NumberEgg,
		Note:           req.Note,
	}

	if err := DB.Create(egg).Error; err != nil {
		log.Printf("Error creating egg record: %v", err)
		return nil, err
	}

	if err := DB.Preload("Coop").First(egg, egg.EggID).Error; err != nil {
		log.Printf("Error reloading created egg record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Egg record created with ID: %d for Coop: %d\n", egg.EggID, egg.CoopID)
	return egg, nil
}

// GetEggByID retrieves an egg record by ID
func GetEggByID(eggID int) (*models.Egg, error) {
	var egg *models.Egg

	if err := DB.Preload("Coop").
		Where("egg_id = ?", eggID).
		First(&egg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("egg record not found")
		}
		log.Printf("Error fetching egg record: %v", err)
		return nil, err
	}

	return egg, nil
}

// GetEggsByCoopID retrieves all egg records for a specific coop
func GetEggsByCoopID(coopID int) ([]models.Egg, error) {
	var eggs []models.Egg

	// Verify that the coop exists
	if _, err := GetCoopByID(coopID); err != nil {
		return nil, fmt.Errorf("coop not found: %v", err)
	}

	if err := DB.Preload("Coop").
		Where("coop_id = ?", coopID).
		Order("date_collect_egg DESC").
		Find(&eggs).Error; err != nil {
		log.Printf("Error fetching egg records: %v", err)
		return nil, err
	}

	return eggs, nil
}

// GetAllEggs retrieves all egg records from the database
func GetAllEggs() ([]models.Egg, error) {
	var eggs []models.Egg

	if err := DB.Preload("Coop").
		Order("date_collect_egg DESC").
		Find(&eggs).Error; err != nil {
		log.Printf("Error fetching all egg records: %v", err)
		return nil, err
	}

	return eggs, nil
}

// UpdateEgg updates an existing egg record
func UpdateEgg(eggID int, req *models.UpdateEggRequest) (*models.Egg, error) {
	egg, err := GetEggByID(eggID)
	if err != nil {
		return nil, err
	}

	// If CoopID is provided, verify it exists
	if req.CoopID > 0 && req.CoopID != egg.CoopID {
		if _, err := GetCoopByID(req.CoopID); err != nil {
			return nil, fmt.Errorf("target coop not found: %v", err)
		}
	}

	// Update only provided fields
	updates := map[string]interface{}{}

	if req.CoopID > 0 {
		updates["coop_id"] = req.CoopID
	}
	if !req.DateCollectEgg.IsZero() {
		updates["date_collect_egg"] = req.DateCollectEgg
	}
	if req.NumberEgg > 0 {
		updates["number_egg"] = req.NumberEgg
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if len(updates) == 0 {
		return egg, nil
	}

	if err := DB.Model(egg).Updates(updates).Error; err != nil {
		log.Printf("Error updating egg record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Egg record %d updated\n", eggID)
	return egg, nil
}

// DeleteEgg deletes an egg record from the database
func DeleteEgg(eggID int) error {
	egg, err := GetEggByID(eggID)
	if err != nil {
		return err
	}

	if err := DB.Delete(egg).Error; err != nil {
		log.Printf("Error deleting egg record: %v", err)
		return err
	}

	fmt.Printf("✓ Egg record %d deleted\n", eggID)
	return nil
}

// DeleteEggsByCoopID deletes all egg records for a specific coop
func DeleteEggsByCoopID(coopID int) error {
	if err := DB.Where("coop_id = ?", coopID).Delete(&models.Egg{}).Error; err != nil {
		log.Printf("Error deleting egg records for coop %d: %v", coopID, err)
		return err
	}

	fmt.Printf("✓ All egg records for coop %d deleted\n", coopID)
	return nil
}
