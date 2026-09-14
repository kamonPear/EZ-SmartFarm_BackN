package models

// FoodDistributionItem is one coop's share within a RecordFoodDistributionRequest
type FoodDistributionItem struct {
	CoopID  int     `json:"coop_id"`
	KgGiven float64 `json:"kg_given"`
}

// RecordFoodDistributionRequest is the payload for saving how a deducted batch of
// food was actually split across coops. The sum of every item's KgGiven must match
// the fixed daily deduction amount for FoodType exactly (see handlers.DailyDeductAmounts).
type RecordFoodDistributionRequest struct {
	FoodType string                 `json:"food_type"`
	Items    []FoodDistributionItem `json:"items"`
}
