package models

import "time"

// CreateVaccineRequest represents the request payload for recording a vaccine administration
type CreateVaccineRequest struct {
	CoopID         int       `json:"coop_id" binding:"required"`
	Name           string    `json:"name" binding:"required"`
	Method         string    `json:"method" binding:"required"`
	RecommendedAge string    `json:"recommended_age"`
	RecordDate     time.Time `json:"record_date" binding:"required"`
	Note           string    `json:"note"`
}
