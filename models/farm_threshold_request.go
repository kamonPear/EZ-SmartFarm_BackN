package models

// UpdateFarmThresholdRequest is the request payload for setting the farm-wide
// target temperature/ammonia level.
type UpdateFarmThresholdRequest struct {
	Temperature float64 `json:"temperature"`
	Ammonia     float64 `json:"ammonia"`
}
