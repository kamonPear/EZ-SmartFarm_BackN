package models

import "time"

// CreateImportFoodRequest represents the request payload for recording a new food import lot.
// Creating an import lot also adds ImportVolume onto the current Foodstock total.
type CreateImportFoodRequest struct {
	ImportVolume int       `json:"import_volume" binding:"required,min=0"`
	ExpiryDate   time.Time `json:"expiry_date" binding:"required"`
}

// ImportFoodResponse represents the response payload for an import food lot
type ImportFoodResponse struct {
	LotID        int       `json:"lot_id"`
	ImportVolume int       `json:"import_volume"`
	ImportDate   time.Time `json:"import_date"`
	ExpiryDate   time.Time `json:"expiry_date"`
}
