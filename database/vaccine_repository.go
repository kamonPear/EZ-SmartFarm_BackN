package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"
)

// GetVaccineByID retrieves a vaccine record by vaccine ID
func GetVaccineByID(vaccineID int) (*models.Vaccine, error) {
	var vaccine models.Vaccine

	result := DB.Where("vaccine_id = ?", vaccineID).First(&vaccine)
	if result.Error != nil {
		log.Printf("Error retrieving vaccine by ID %d: %v", vaccineID, result.Error)
		return nil, result.Error
	}

	return &vaccine, nil
}

// GetVaccinesByCoopID retrieves all vaccine records for a specific coop
func GetVaccinesByCoopID(coopID int) ([]models.Vaccine, error) {
	var vaccines []models.Vaccine

	result := DB.Where("coop_id = ?", coopID).Find(&vaccines)
	if result.Error != nil {
		log.Printf("Error retrieving vaccines for coop ID %d: %v", coopID, result.Error)
		return nil, result.Error
	}

	return vaccines, nil
}

// GetAllVaccines retrieves all vaccine records from database
func GetAllVaccines() ([]models.Vaccine, error) {
	var vaccines []models.Vaccine

	result := DB.Find(&vaccines)
	if result.Error != nil {
		log.Printf("Error retrieving all vaccines: %v", result.Error)
		return nil, result.Error
	}

	return vaccines, nil
}

// DeleteVaccine deletes a vaccine record by vaccine ID
func DeleteVaccine(vaccineID int) error {
	result := DB.Where("vaccine_id = ?", vaccineID).Delete(&models.Vaccine{})
	if result.Error != nil {
		log.Printf("Error deleting vaccine ID %d: %v", vaccineID, result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("No vaccine found with ID %d", vaccineID)
		return fmt.Errorf("no vaccine found with ID %d", vaccineID)
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


