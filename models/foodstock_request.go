package models

import "time"

// CreateFoodstockRequest represents the request payload for creating foodstock
type CreateFoodstockRequest struct {
	QuantityCurrent float64   `json:"quantity_current" binding:"required,min=0"`
	MinQuantity     float64   `json:"min_quantity" binding:"min=0"`
	ImportDate      time.Time `json:"import_date" binding:"required"`
	ExpiryDate      time.Time `json:"expiry_date" binding:"required"`
	DateUp          time.Time `json:"date_up" binding:"required"`
}

// UpdateFoodstockRequest represents the request payload for updating foodstock
type UpdateFoodstockRequest struct {
	QuantityCurrent float64   `json:"quantity_current" binding:"min=0"`
	MinQuantity     float64   `json:"min_quantity" binding:"min=0"`
	ImportDate      time.Time `json:"import_date"`
	ExpiryDate      time.Time `json:"expiry_date"`
	DateUp          time.Time `json:"date_up"`
}

// FoodstockResponse represents the response payload for foodstock data
type FoodstockResponse struct {
	FoodID          int       `json:"food_id"`
	QuantityCurrent float64   `json:"quantity_current"`
	ImportDate      time.Time `json:"import_date"`
	ExpiryDate      time.Time `json:"expiry_date"`
	DateUp          time.Time `json:"date_up"`
}
