package database

import (
	"fmt"
	"log"
	"strings"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// IsDuplicateEggDate reports whether err is a unique-constraint violation on
// (coop_id, date_collect_egg) — i.e. an egg record already exists for that
// coop on that date (schema.sql: UNIQUE (coop_id, date_collect_egg))
func IsDuplicateEggDate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") && strings.Contains(msg, "coop_id")
}

// CreateEgg creates a new egg record in the database, only if req.CoopID belongs to userID
func CreateEgg(req *models.CreateEggRequest, userID int) (*models.Egg, error) {
	owned, err := CoopBelongsToUser(req.CoopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
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

// GetEggByID retrieves an egg record by ID, only if its coop belongs to userID
func GetEggByID(eggID int, userID int) (*models.Egg, error) {
	var egg *models.Egg

	if err := DB.Preload("Coop").
		Where("egg_id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", eggID, userID).
		First(&egg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error fetching egg record: %v", err)
		return nil, err
	}

	return egg, nil
}

// GetEggsByCoopID retrieves all egg records for a specific coop, only if it belongs to userID
func GetEggsByCoopID(coopID int, userID int) ([]models.Egg, error) {
	var eggs []models.Egg

	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
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

// GetAllEggs retrieves every egg record belonging to any of userID's coops
func GetAllEggs(userID int) ([]models.Egg, error) {
	var eggs []models.Egg

	if err := DB.Preload("Coop").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Order("date_collect_egg DESC").
		Find(&eggs).Error; err != nil {
		log.Printf("Error fetching all egg records: %v", err)
		return nil, err
	}

	return eggs, nil
}

// UpdateEgg updates an existing egg record, only if it (and any target coop) belongs to userID
func UpdateEgg(eggID int, req *models.UpdateEggRequest, userID int) (*models.Egg, error) {
	egg, err := GetEggByID(eggID, userID)
	if err != nil {
		return nil, err
	}

	// If CoopID is provided, verify the target coop exists and belongs to userID too
	if req.CoopID > 0 && req.CoopID != egg.CoopID {
		owned, err := CoopBelongsToUser(req.CoopID, userID)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, ErrNotFound
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

// DeleteEgg deletes an egg record from the database, only if its coop belongs to userID
func DeleteEgg(eggID int, userID int) error {
	egg, err := GetEggByID(eggID, userID)
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
