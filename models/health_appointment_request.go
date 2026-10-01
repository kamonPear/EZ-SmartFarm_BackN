package models

import "time"

// CreateHealthAppointmentRequest is the request payload for booking a manual
// health-check appointment. AppointmentDate is decoded straight from an RFC3339
// string (e.g. "2026-10-05T00:00:00Z"), matching how the frontend already sends
// record_date for health/egg/vaccine records.
type CreateHealthAppointmentRequest struct {
	CoopID          int       `json:"coop_id"`
	AppointmentDate time.Time `json:"appointment_date"`
}
