package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

func normalizeHealthDate(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

// CreateHealth creates a new health record in the database
func CreateHealth(req *models.CreateHealthRequest) (*models.Health, error) {
	// Verify coop exists
	if _, err := GetCoopByID(req.CoopID); err != nil {
		return nil, fmt.Errorf("coop not found: %v", err)
	}

	normalizedDate := normalizeHealthDate(req.RecordDate)

	health := &models.Health{
		CoopID:     req.CoopID,
		Healthy:    req.Healthy,
		PoorHealth: req.PoorHealth,
		Note:       req.Note,
		RecordDate: normalizedDate,
	}

	if err := DB.Create(health).Error; err != nil {
		log.Printf("Error creating health record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Health record created with ID: %d for Coop: %d\n", health.HealthID, health.CoopID)
	return health, nil
}

// GetHealthByID retrieves a health record by ID
func GetHealthByID(id int) (*models.Health, error) {
	var h *models.Health

	if err := DB.Preload("Coop").Where("health_id = ?", id).First(&h).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("health record not found")
		}
		log.Printf("Error fetching health record: %v", err)
		return nil, err
	}

	return h, nil
}

// GetHealthsByCoopID retrieves all health records for a specific coop
func GetHealthsByCoopID(coopID int) ([]models.Health, error) {
	var hs []models.Health

	// Verify coop exists
	if _, err := GetCoopByID(coopID); err != nil {
		return nil, fmt.Errorf("coop not found: %v", err)
	}

	if err := DB.Preload("Coop").Where("coop_id = ?", coopID).Order("record_date DESC").Find(&hs).Error; err != nil {
		log.Printf("Error fetching health records: %v", err)
		return nil, err
	}

	return hs, nil
}

// GetAllHealths retrieves all health records
func GetAllHealths() ([]models.Health, error) {
	var hs []models.Health

	if err := DB.Preload("Coop").Order("record_date DESC").Find(&hs).Error; err != nil {
		log.Printf("Error fetching all health records: %v", err)
		return nil, err
	}

	return hs, nil
}

// UpdateHealth updates an existing health record
func UpdateHealth(id int, req *models.UpdateHealthRequest) (*models.Health, error) {
	h, err := GetHealthByID(id)
	if err != nil {
		return nil, err
	}

	// if coop change provided, validate
	if req.CoopID > 0 && req.CoopID != h.CoopID {
		if _, err := GetCoopByID(req.CoopID); err != nil {
			return nil, fmt.Errorf("target coop not found: %v", err)
		}
	}

	updates := map[string]interface{}{}
	if req.CoopID > 0 {
		updates["coop_id"] = req.CoopID
	}
	if req.Healthy >= 0 {
		updates["healthy"] = req.Healthy
	}
	if req.PoorHealth >= 0 {
		updates["poor_health"] = req.PoorHealth
	}
	if !req.RecordDate.IsZero() {
		updates["record_date"] = normalizeHealthDate(req.RecordDate)
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if len(updates) == 0 {
		return h, nil
	}

	if err := DB.Model(h).Updates(updates).Error; err != nil {
		log.Printf("Error updating health record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Health record %d updated\n", id)
	return h, nil
}

// DeleteHealth deletes a health record
func DeleteHealth(id int) error {
	h, err := GetHealthByID(id)
	if err != nil {
		return err
	}

	if err := DB.Delete(h).Error; err != nil {
		log.Printf("Error deleting health record: %v", err)
		return err
	}

	fmt.Printf("✓ Health record %d deleted\n", id)
	return nil
}

// DeleteHealthsByCoopID deletes all health records for a coop
func DeleteHealthsByCoopID(coopID int) error {
	if err := DB.Where("coop_id = ?", coopID).Delete(&models.Health{}).Error; err != nil {
		log.Printf("Error deleting health records for coop %d: %v", coopID, err)
		return err
	}

	fmt.Printf("✓ All health records for coop %d deleted\n", coopID)
	return nil
}
