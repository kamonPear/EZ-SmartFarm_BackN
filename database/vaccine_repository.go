package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateVaccine creates a new vaccine administration record for a coop.
// The coop must already be loaded (via a caller that verified ownership) so
// name_coop/birthday can be copied onto the vaccine row to satisfy the
// fk_name_coop_vaccines constraint.
func CreateVaccine(req *models.CreateVaccineRequest, coop *models.Coop) (*models.Vaccine, error) {
	coopID := req.CoopID
	recordDate := req.RecordDate
	vaccine := models.Vaccine{
		CoopID:         &coopID,
		NameCoop:       coop.NameCoop,
		Birthday:       &coop.Birthday,
		Name:           req.Name,
		RecordDate:     &recordDate,
		Method:         req.Method,
		RecommendedAge: req.RecommendedAge,
		Note:           req.Note,
	}

	if err := DB.Create(&vaccine).Error; err != nil {
		log.Printf("Error creating vaccine: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Vaccine created with ID: %d\n", vaccine.VaccineID)
	return &vaccine, nil
}

// GetVaccineByID retrieves a vaccine record by vaccine ID, only if its coop belongs to userID
func GetVaccineByID(vaccineID int, userID int) (*models.Vaccine, error) {
	var vaccine models.Vaccine

	result := DB.Where("vaccine_id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", vaccineID, userID).First(&vaccine)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error retrieving vaccine by ID %d: %v", vaccineID, result.Error)
		return nil, result.Error
	}

	return &vaccine, nil
}

// GetVaccinesByCoopID retrieves all vaccine records for a specific coop, only if it belongs to userID
func GetVaccinesByCoopID(coopID int, userID int) ([]models.Vaccine, error) {
	var vaccines []models.Vaccine

	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	result := DB.Where("coop_id = ?", coopID).Find(&vaccines)
	if result.Error != nil {
		log.Printf("Error retrieving vaccines for coop ID %d: %v", coopID, result.Error)
		return nil, result.Error
	}

	return vaccines, nil
}

// GetAllVaccines retrieves every vaccine record belonging to any of userID's coops
func GetAllVaccines(userID int) ([]models.Vaccine, error) {
	var vaccines []models.Vaccine

	err := DB.Preload("Coop").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Find(&vaccines).Error

	if err != nil {
		log.Printf("Error retrieving all vaccines: %v", err)
	}

	return vaccines, err
}

// UpdateVaccine updates an existing vaccine record, only if its coop belongs to userID.
// Fields left zero-valued on req are left unchanged. Uses load-mutate-Save (not a map
// .Updates()) so the Vaccine.BeforeSave hook re-syncs the legacy "name" column from the
// actual updated Name instead of from a zero-valued receiver struct.
func UpdateVaccine(id int, req *models.UpdateVaccineRequest, userID int) (*models.Vaccine, error) {
	vaccine, err := GetVaccineByID(id, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		vaccine.Name = req.Name
	}
	if req.Method != "" {
		vaccine.Method = req.Method
	}
	if req.RecommendedAge != "" {
		vaccine.RecommendedAge = req.RecommendedAge
	}
	if !req.RecordDate.IsZero() {
		recordDate := req.RecordDate
		vaccine.RecordDate = &recordDate
	}
	if req.Note != "" {
		vaccine.Note = req.Note
	}

	if err := DB.Save(vaccine).Error; err != nil {
		log.Printf("Error updating vaccine ID %d: %v", id, err)
		return nil, err
	}

	return vaccine, nil
}

// DeleteVaccine deletes a vaccine record by vaccine ID, only if its coop belongs to userID
func DeleteVaccine(vaccineID int, userID int) error {
	vaccine, err := GetVaccineByID(vaccineID, userID)
	if err != nil {
		return err
	}

	result := DB.Delete(vaccine)
	if result.Error != nil {
		log.Printf("Error deleting vaccine ID %d: %v", vaccineID, result.Error)
		return result.Error
	}

	log.Printf("Successfully deleted vaccine ID %d", vaccineID)
	return nil
}

// DeleteVaccinesByCoopID deletes all vaccine records for a specific coop
func DeleteVaccinesByCoopID(coopID int) error {
	result := DB.Where("coop_id = ?", coopID).Delete(&models.Vaccine{})
	if result.Error != nil {
		log.Printf("Error deleting vaccines for coop ID %d: %v", coopID, result.Error)
		return result.Error
	}

	log.Printf("Successfully deleted %d vaccine records for coop ID %d", result.RowsAffected, coopID)
	return nil
}

// DeleteVaccinesByCoopIDForUser deletes all vaccine records for a coop, only if it belongs to userID
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

