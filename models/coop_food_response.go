package models

// CoopFoodConsumption represents the estimated daily food consumption of a single coop.
// The estimate is derived from the chickens' age (which decides FoodType) and the coop's
// share of Amount (chicken headcount) among every coop currently on that same FoodType -
// there is no per-coop feed sensor, so this is a proportional split of the farm-wide daily
// deduction (see handlers.DailyDeductAmounts) rather than a measured value.
type CoopFoodConsumption struct {
	CoopID            int     `json:"coop_id"`
	NameCoop          string  `json:"name_coop"`
	Amount            int     `json:"amount"`
	AgeWeeks          int     `json:"age_weeks"`
	FoodType          string  `json:"food_type"`
	EstimatedKgPerDay float64 `json:"estimated_kg_per_day"`
}
