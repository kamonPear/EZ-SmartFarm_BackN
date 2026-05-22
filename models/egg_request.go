package models

import "time"

// CreateEggRequest represents the request payload for creating an egg record
type CreateEggRequest struct {
	CoopID         int       `json:"coop_id" binding:"required"`
	DateCollectEgg time.Time `json:"date_collect_egg" binding:"required"`
	NumberEgg      int       `json:"number_egg" binding:"required,min=1"`
	Note           string    `json:"note"`
}

// UpdateEggRequest represents the request payload for updating an egg record
type UpdateEggRequest struct {
	CoopID         int       `json:"coop_id"`
	DateCollectEgg time.Time `json:"date_collect_egg"`
	NumberEgg      int       `json:"number_egg" binding:"min=1"`
	Note           string    `json:"note"`
}

// EggResponse represents the response payload for egg data
type EggResponse struct {
	EggID          int       `json:"egg_id"`
	CoopID         int       `json:"coop_id"`
	DateCollectEgg time.Time `json:"date_collect_egg"`
	NumberEgg      int       `json:"number_egg"`
	Note           string    `json:"note"`
}
