package models

import "time"

// UpdateFoodstockRequest represents the request payload for manually correcting the current stock total
type UpdateFoodstockRequest struct {
	QuantityCurrent float64   `json:"quantity_current" binding:"min=0"`
	DateUp          time.Time `json:"date_up"`
}

// FoodstockResponse represents the response payload for foodstock data
type FoodstockResponse struct {
	FoodID          int       `json:"food_id"`
	FoodType        string    `json:"food_type"`
	QuantityCurrent float64   `json:"quantity_current"`
	DateUp          time.Time `json:"date_up"`
}

// DeductFoodstockRequest represents the request payload for cutting stock of a chosen food type
type DeductFoodstockRequest struct {
	FoodType string `json:"food_type" binding:"required"`
}
