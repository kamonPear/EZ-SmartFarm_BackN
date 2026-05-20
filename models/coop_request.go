package models

import "time"

// CreateCoopRequest represents the request payload for creating a coop
type CreateCoopRequest struct {
	DateAdoptAnimals time.Time `json:"date_adopt_animals" binding:"required"`
	Amount           int       `json:"amount" binding:"required,min=1"`
	Birthday         time.Time `json:"birthday" binding:"required"`
	Note             string    `json:"note"`
}

// UpdateCoopRequest represents the request payload for updating a coop
type UpdateCoopRequest struct {
	DateAdoptAnimals time.Time `json:"date_adopt_animals"`
	Amount           int       `json:"amount" binding:"min=1"`
	Birthday         time.Time `json:"birthday"`
	Note             string    `json:"note"`
}

// CoopResponse represents the response payload for coop data
type CoopResponse struct {
	CoopID           int       `json:"coop_id"`
	DateAdoptAnimals time.Time `json:"date_adopt_animals"`
	Amount           int       `json:"amount"`
	Birthday         time.Time `json:"birthday"`
	Note             string    `json:"note"`
}
