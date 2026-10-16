package models

// FoodAgeConfig holds, per user, the feeding rules the farm owner sets for themselves:
//   - which chicken age range (in weeks, counted from a coop's birthday) is fed which
//     food type, and
//   - how many grams one bird of that range eats per day.
//
// Nothing here is hardcoded: until the owner saves a config there is no row, Configured
// is false and no coop is assigned a food type or a consumption amount. Each age range
// is [min, max): min inclusive, max exclusive; a max of 0 means "no upper limit".
type FoodAgeConfig struct {
	ID                int     `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID            int     `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_food_age_config_user" json:"user_id"`
	SmallMinWeeks     int     `gorm:"column:small_min_weeks;type:int;not null" json:"small_min_weeks"`
	SmallMaxWeeks     int     `gorm:"column:small_max_weeks;type:int;not null" json:"small_max_weeks"`
	LargeMinWeeks     int     `gorm:"column:large_min_weeks;type:int;not null" json:"large_min_weeks"`
	LargeMaxWeeks     int     `gorm:"column:large_max_weeks;type:int;not null" json:"large_max_weeks"`
	SmallGramsPerBird float64 `gorm:"column:small_grams_per_bird;type:decimal(8,2);not null;default:0" json:"small_grams_per_bird"`
	LargeGramsPerBird float64 `gorm:"column:large_grams_per_bird;type:decimal(8,2);not null;default:0" json:"large_grams_per_bird"`
	// Configured is false when the owner has never saved a config. Not stored in the table.
	Configured bool `gorm:"-" json:"configured"`
}

// TableName specifies the table name for FoodAgeConfig model
func (FoodAgeConfig) TableName() string {
	return "food_age_config"
}

// UpdateFoodAgeConfigRequest is the request payload for saving the rules.
type UpdateFoodAgeConfigRequest struct {
	SmallMinWeeks     int     `json:"small_min_weeks"`
	SmallMaxWeeks     int     `json:"small_max_weeks"`
	LargeMinWeeks     int     `json:"large_min_weeks"`
	LargeMaxWeeks     int     `json:"large_max_weeks"`
	SmallGramsPerBird float64 `json:"small_grams_per_bird"`
	LargeGramsPerBird float64 `json:"large_grams_per_bird"`
}

func inAgeRange(age, min, max int) bool {
	return age >= min && (max == 0 || age < max)
}

// FoodTypeForAge returns the food type whose configured range contains ageWeeks, or ""
// if the owner hasn't configured anything or the age falls in neither range (that coop
// is then simply not fed/deducted).
func (c FoodAgeConfig) FoodTypeForAge(ageWeeks int) string {
	if !c.Configured {
		return ""
	}
	if inAgeRange(ageWeeks, c.SmallMinWeeks, c.SmallMaxWeeks) {
		return FoodTypeSmallPellet
	}
	if inAgeRange(ageWeeks, c.LargeMinWeeks, c.LargeMaxWeeks) {
		return FoodTypeLargePellet
	}
	return ""
}

// GramsPerBirdFor returns the configured grams/bird/day for foodType (0 if unknown).
func (c FoodAgeConfig) GramsPerBirdFor(foodType string) float64 {
	switch foodType {
	case FoodTypeSmallPellet:
		return c.SmallGramsPerBird
	case FoodTypeLargePellet:
		return c.LargeGramsPerBird
	}
	return 0
}

// AgeRangeFor returns the configured [min, max) week range for foodType (max 0 = no limit).
func (c FoodAgeConfig) AgeRangeFor(foodType string) (min, max int) {
	switch foodType {
	case FoodTypeSmallPellet:
		return c.SmallMinWeeks, c.SmallMaxWeeks
	case FoodTypeLargePellet:
		return c.LargeMinWeeks, c.LargeMaxWeeks
	}
	return 0, 0
}

// CoopFeed is the single calculation behind every food number: given a coop's age in
// weeks and its headcount, it returns the food type that age range is fed and the kg that
// whole coop eats per day (headcount x configured grams per bird / 1000), all read from
// the owner's saved config. Returns ("", 0) if the age matches no configured range.
func (c FoodAgeConfig) CoopFeed(ageWeeks, headcount int) (foodType string, kgPerDay float64) {
	foodType = c.FoodTypeForAge(ageWeeks)
	if foodType == "" || headcount <= 0 {
		return foodType, 0
	}
	return foodType, float64(headcount) * c.GramsPerBirdFor(foodType) / 1000
}

// Validate checks the rules are sane: non-negative ages, small max > small min, large max
// (when set) > large min, ranges don't overlap, and grams per bird are positive.
// Returns "" when valid.
func (r UpdateFoodAgeConfigRequest) Validate() string {
	if r.SmallMinWeeks < 0 || r.SmallMaxWeeks < 0 || r.LargeMinWeeks < 0 || r.LargeMaxWeeks < 0 {
		return "อายุต้องไม่ติดลบ"
	}
	if r.SmallMaxWeeks == 0 || r.SmallMaxWeeks <= r.SmallMinWeeks {
		return "อายุสิ้นสุดของเม็ดเล็กต้องมากกว่าอายุเริ่มต้น"
	}
	if r.LargeMaxWeeks != 0 && r.LargeMaxWeeks <= r.LargeMinWeeks {
		return "อายุสิ้นสุดของเม็ดใหญ่ต้องมากกว่าอายุเริ่มต้น (หรือเว้นเป็น 0 = ไม่จำกัด)"
	}
	smallEndsBeforeLarge := r.SmallMaxWeeks <= r.LargeMinWeeks
	largeEndsBeforeSmall := r.LargeMaxWeeks != 0 && r.LargeMaxWeeks <= r.SmallMinWeeks
	if !smallEndsBeforeLarge && !largeEndsBeforeSmall {
		return "ช่วงอายุของเม็ดเล็กและเม็ดใหญ่ต้องไม่ซ้อนทับกัน"
	}
	if r.SmallGramsPerBird <= 0 || r.LargeGramsPerBird <= 0 {
		return "ปริมาณอาหารต่อตัวต่อวัน (กรัม) ต้องมากกว่า 0"
	}
	return ""
}
