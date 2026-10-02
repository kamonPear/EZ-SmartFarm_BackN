package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateVaccine creates a new vaccine administration record for a coop.
// The coop must already be loaded (via a caller that verified ownership) so
// name_coop/birthday can be copied onto the history row.
func CreateVaccine(req *models.CreateVaccineRequest, coop *models.Coop) (*models.VaccineHistory, error) {
	history := models.VaccineHistory{
		CoopID:         req.CoopID,
		NameCoop:       coop.NameCoop,
		Birthday:       coop.Birthday,
		Name:           req.Name,
		RecordDate:     req.RecordDate,
		Method:         req.Method,
		RecommendedAge: req.RecommendedAge,
		Note:           req.Note,
	}

	if err := DB.Create(&history).Error; err != nil {
		log.Printf("Error creating vaccine history record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Vaccine history record created with ID: %d\n", history.VaccineID)
	return &history, nil
}

// GetVaccineByID retrieves a vaccine history record by ID, only if its coop belongs to userID
func GetVaccineByID(vaccineID int, userID int) (*models.VaccineHistory, error) {
	var history models.VaccineHistory

	result := DB.Where("id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", vaccineID, userID).First(&history)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error retrieving vaccine history by ID %d: %v", vaccineID, result.Error)
		return nil, result.Error
	}

	return &history, nil
}

// GetVaccinesByCoopID retrieves all vaccine history records for a specific coop, only if it belongs to userID
func GetVaccinesByCoopID(coopID int, userID int) ([]models.VaccineHistory, error) {
	var history []models.VaccineHistory

	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	result := DB.Where("coop_id = ?", coopID).Find(&history)
	if result.Error != nil {
		log.Printf("Error retrieving vaccine history for coop ID %d: %v", coopID, result.Error)
		return nil, result.Error
	}

	return history, nil
}

// GetAllVaccines retrieves every vaccine history record belonging to any of userID's coops
func GetAllVaccines(userID int) ([]models.VaccineHistory, error) {
	var history []models.VaccineHistory

	err := DB.Preload("Coop").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Find(&history).Error

	if err != nil {
		log.Printf("Error retrieving all vaccine history: %v", err)
	}

	return history, err
}

// UpdateVaccine updates an existing vaccine history record, only if its coop belongs to userID.
// Fields left zero-valued on req are left unchanged.
func UpdateVaccine(id int, req *models.UpdateVaccineRequest, userID int) (*models.VaccineHistory, error) {
	history, err := GetVaccineByID(id, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		history.Name = req.Name
	}
	if req.Method != "" {
		history.Method = req.Method
	}
	if req.RecommendedAge != "" {
		history.RecommendedAge = req.RecommendedAge
	}
	if !req.RecordDate.IsZero() {
		history.RecordDate = req.RecordDate
	}
	if req.Note != "" {
		history.Note = req.Note
	}

	if err := DB.Save(history).Error; err != nil {
		log.Printf("Error updating vaccine history ID %d: %v", id, err)
		return nil, err
	}

	return history, nil
}

// DeleteVaccine deletes a vaccine history record by ID, only if its coop belongs to userID
func DeleteVaccine(vaccineID int, userID int) error {
	history, err := GetVaccineByID(vaccineID, userID)
	if err != nil {
		return err
	}

	result := DB.Delete(history)
	if result.Error != nil {
		log.Printf("Error deleting vaccine history ID %d: %v", vaccineID, result.Error)
		return result.Error
	}

	log.Printf("Successfully deleted vaccine history ID %d", vaccineID)
	return nil
}

// DeleteVaccinesByCoopID deletes all vaccine history records for a specific coop
func DeleteVaccinesByCoopID(coopID int) error {
	result := DB.Where("coop_id = ?", coopID).Delete(&models.VaccineHistory{})
	if result.Error != nil {
		log.Printf("Error deleting vaccine history for coop ID %d: %v", coopID, result.Error)
		return result.Error
	}

	log.Printf("Successfully deleted %d vaccine history records for coop ID %d", result.RowsAffected, coopID)
	return nil
}

// DeleteVaccinesByCoopIDForUser deletes all vaccine history records for a coop, only if it belongs to userID
func DeleteVaccinesByCoopIDForUser(coopID int, userID int) error {
	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	return DeleteVaccinesByCoopID(coopID)
}
