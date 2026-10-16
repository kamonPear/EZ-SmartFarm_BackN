package models

// FoodAgeConfig holds, per user, which chicken age range (in weeks, from a coop's
// birthday) is fed which food type. Replaces the old hardcoded "small pellet below 14
// weeks, large pellet from 14 weeks" rule. Each range is [min, max): min inclusive,
// max exclusive; a max of 0 means "no upper limit". A row is only created the first time
// the owner saves a config - until then DefaultFoodAgeConfig (the old 0-14 / 14+ rule)
// applies.
type FoodAgeConfig struct {
	ID            int `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID        int `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_food_age_config_user" json:"user_id"`
	SmallMinWeeks int `gorm:"column:small_min_weeks;type:int;not null" json:"small_min_weeks"`
	SmallMaxWeeks int `gorm:"column:small_max_weeks;type:int;not null" json:"small_max_weeks"`
	LargeMinWeeks int `gorm:"column:large_min_weeks;type:int;not null" json:"large_min_weeks"`
	LargeMaxWeeks int `gorm:"column:large_max_weeks;type:int;not null" json:"large_max_weeks"`
}

// TableName specifies the table name for FoodAgeConfig model
func (FoodAgeConfig) TableName() string {
	return "food_age_config"
}

// UpdateFoodAgeConfigRequest is the request payload for saving the age ranges.
type UpdateFoodAgeConfigRequest struct {
	SmallMinWeeks int `json:"small_min_weeks"`
	SmallMaxWeeks int `json:"small_max_weeks"`
	LargeMinWeeks int `json:"large_min_weeks"`
	LargeMaxWeeks int `json:"large_max_weeks"`
}

// DefaultFoodAgeConfig is the original fixed rule (small 0-14 weeks, large 14+ weeks),
// used for users who haven't configured anything.
func DefaultFoodAgeConfig(userID int) FoodAgeConfig {
	return FoodAgeConfig{
		UserID:        userID,
		SmallMinWeeks: 0,
		SmallMaxWeeks: AgeThresholdWeeksSmallToLarge,
		LargeMinWeeks: AgeThresholdWeeksSmallToLarge,
		LargeMaxWeeks: 0,
	}
}

func inAgeRange(age, min, max int) bool {
	return age >= min && (max == 0 || age < max)
}

// FoodTypeForAge returns the food type whose configured range contains ageWeeks, or ""
// if the age falls in neither range (that coop is then simply not fed/deducted).
func (c FoodAgeConfig) FoodTypeForAge(ageWeeks int) string {
	if inAgeRange(ageWeeks, c.SmallMinWeeks, c.SmallMaxWeeks) {
		return FoodTypeSmallPellet
	}
	if inAgeRange(ageWeeks, c.LargeMinWeeks, c.LargeMaxWeeks) {
		return FoodTypeLargePellet
	}
	return ""
}

// Validate checks the ranges are sane: non-negative, small max > small min, large max
// (when set) > large min, and the two ranges don't overlap. Returns "" when valid.
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
	return ""
}
