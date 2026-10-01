package database

import (
	"errors"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

func normalizeAppointmentDate(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

// CreateHealthAppointment books a manual health-check appointment for a coop, only
// if req.CoopID belongs to userID. Returns the existing row (not an error) if an
// appointment for the same coop+date already exists - matches the idempotent
// de-dupe behavior the frontend's old localStorage version had.
func CreateHealthAppointment(req *models.CreateHealthAppointmentRequest, userID int) (*models.HealthAppointment, error) {
	owned, err := CoopBelongsToUser(req.CoopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	date := normalizeAppointmentDate(req.AppointmentDate)

	var existing models.HealthAppointment
	if err := DB.Where("coop_id = ? AND appointment_date = ?", req.CoopID, date).First(&existing).Error; err == nil {
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error checking existing health appointment: %v", err)
		return nil, err
	}

	appt := &models.HealthAppointment{
		CoopID:          req.CoopID,
		AppointmentDate: date,
	}
	if err := DB.Create(appt).Error; err != nil {
		log.Printf("Error creating health appointment: %v", err)
		return nil, err
	}

	log.Printf("✓ Health appointment created with ID: %d for Coop: %d on %s", appt.AppointmentID, appt.CoopID, date.Format("2006-01-02"))
	return appt, nil
}

// GetHealthAppointments returns every today-or-later manual appointment belonging to
// userID's coops (optionally filtered to one coop), soonest first. Past appointments
// are left in the table (harmless history) but excluded here - once the date
// arrives it becomes a normal health check, not an upcoming "appointment" anymore.
func GetHealthAppointments(userID int, coopID *int) ([]models.HealthAppointment, error) {
	var appts []models.HealthAppointment
	today := normalizeAppointmentDate(time.Now())

	q := DB.Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Where("appointment_date >= ?", today)
	if coopID != nil {
		q = q.Where("coop_id = ?", *coopID)
	}
	if err := q.Order("appointment_date ASC").Find(&appts).Error; err != nil {
		log.Printf("Error fetching health appointments: %v", err)
		return nil, err
	}
	return appts, nil
}

// DeleteHealthAppointment removes a manual appointment, only if its coop belongs to userID.
func DeleteHealthAppointment(id int, userID int) error {
	var appt models.HealthAppointment
	err := DB.Where("appointment_id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", id, userID).
		First(&appt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		log.Printf("Error fetching health appointment to delete: %v", err)
		return err
	}

	if err := DB.Delete(&appt).Error; err != nil {
		log.Printf("Error deleting health appointment: %v", err)
		return err
	}

	log.Printf("✓ Health appointment %d deleted", id)
	return nil
}
