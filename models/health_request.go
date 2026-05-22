package models

import "time"

// CreateHealthRequest represents the request payload for creating a health record
type CreateHealthRequest struct {
	CoopID     int       `json:"coop_id" binding:"required"`
	Healthy    int       `json:"healthy" binding:"required"`
	PoorHealth int       `json:"poor_health" binding:"required"`
	RecordDate time.Time `json:"record_date" binding:"required"`
	Note       string    `json:"note"`
}

// UpdateHealthRequest represents the payload for updating a health record
type UpdateHealthRequest struct {
	CoopID     int       `json:"coop_id"`
	Healthy    int       `json:"healthy"`
	PoorHealth int       `json:"poor_health"`
	RecordDate time.Time `json:"record_date"`
	Note       string    `json:"note"`
}

// HealthResponse represents the response payload for health data
type HealthResponse struct {
	HealthID   int       `json:"health_id"`
	CoopID     int       `json:"coop_id"`
	Healthy    int       `json:"healthy"`
	PoorHealth int       `json:"poor_health"`
	RecordDate time.Time `json:"record_date"`
	Note       string    `json:"note"`
}
