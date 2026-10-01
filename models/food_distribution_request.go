package models

// FoodDistributionItem is one coop's share within a RecordFoodDistributionRequest
type FoodDistributionItem struct {
	CoopID  int     `json:"coop_id"`
	KgGiven float64 `json:"kg_given"`
}

// RecordFoodDistributionRequest is the payload for a manual stock cut: it both deducts
// foodstock and records how it was split across coops in one action. The deducted total
// is exactly the sum of every item's KgGiven (there is no fixed amount to match).
type RecordFoodDistributionRequest struct {
	FoodType string                 `json:"food_type"`
	Items    []FoodDistributionItem `json:"items"`
}
