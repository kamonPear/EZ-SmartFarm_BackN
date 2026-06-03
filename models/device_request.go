package models

import "time"

// CreateDeviceRequest represents the request payload for creating a device
type CreateDeviceRequest struct {
	CoopID        int    `json:"coop_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	DeviceType    string `json:"device_type" binding:"required"`
	CurrentStatus string `json:"current_status"`
}

// UpdateDeviceRequest represents the request payload for updating a device
type UpdateDeviceRequest struct {
	Name          string `json:"name"`
	DeviceType    string `json:"device_type"`
	CurrentStatus string `json:"current_status"`
}

// DeviceResponse represents the response payload for device data
type DeviceResponse struct {
	DeviceID      int       `json:"device_id"`
	CoopID        int       `json:"coop_id"`
	Name          string    `json:"name"`
	DeviceType    string    `json:"device_type"`
	CurrentStatus string    `json:"current_status"`
	LastUpdate    time.Time `json:"last_update"`
}
