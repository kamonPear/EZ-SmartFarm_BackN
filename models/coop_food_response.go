package models

// CoopFoodConsumption represents the estimated daily food consumption of a single coop.
// The estimate is derived from the chickens' age (which decides FoodType) and the coop's
// own headcount (Amount x handlers.FoodConsumptionKgPerBirdPerDay[FoodType]) - there is
// no per-coop feed sensor, so this is still an estimate, not a measured value, but it is
// the same formula actually used to deduct stock (handlers.ComputeDailyFoodConsumption).
type CoopFoodConsumption struct {
	CoopID            int     `json:"coop_id"`
	NameCoop          string  `json:"name_coop"`
	Amount            int     `json:"amount"`
	AgeWeeks          int     `json:"age_weeks"`
	FoodType          string  `json:"food_type"`
	EstimatedKgPerDay float64 `json:"estimated_kg_per_day"`
}
