package models

import "time"

// UpdateFoodstockRequest represents the request payload for manually correcting the current stock total
type UpdateFoodstockRequest struct {
	QuantityCurrent float64   `json:"quantity_current" binding:"min=0"`
	DateUp          time.Time `json:"date_up"`
}

// UpdateFoodMinQuantityRequest represents the request payload for setting the low-stock threshold of one food type
type UpdateFoodMinQuantityRequest struct {
	FoodType    string  `json:"food_type"`
	MinQuantity float64 `json:"min_quantity"`
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
