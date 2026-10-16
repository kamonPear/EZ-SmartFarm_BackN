package models

import "sort"

// FoodAgeRule is one row of the feeding table the farm owner fills in for themselves:
// "chickens aged [MinWeeks, MaxWeeks) are fed FoodType, and one bird eats GramsPerBird
// grams per day". An owner can add as many rows as they like (several age brackets per
// food type, each with its own grams). MaxWeeks 0 means "no upper limit".
//
// Nothing is hardcoded: a coop's food type and daily consumption are looked up from these
// rows by its age (counted from its birthday). No matching row = no food type, no
// consumption, nothing deducted for that coop.
type FoodAgeRule struct {
	ID           int     `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID       int     `gorm:"column:user_id;type:int;size:32;index:idx_food_age_rule_user" json:"user_id"`
	FoodType     string  `gorm:"column:food_type;type:varchar(20);not null" json:"food_type"`
	MinWeeks     int     `gorm:"column:min_weeks;type:int;not null" json:"min_weeks"`
	MaxWeeks     int     `gorm:"column:max_weeks;type:int;not null;default:0" json:"max_weeks"`
	GramsPerBird float64 `gorm:"column:grams_per_bird;type:decimal(10,3);not null" json:"grams_per_bird"`
}

// TableName specifies the table name for FoodAgeRule model
func (FoodAgeRule) TableName() string {
	return "food_age_rule"
}

// FoodAgeRuleInput is one row of the save request (the owner's table is replaced whole).
type FoodAgeRuleInput struct {
	FoodType     string  `json:"food_type"`
	MinWeeks     int     `json:"min_weeks"`
	MaxWeeks     int     `json:"max_weeks"`
	GramsPerBird float64 `json:"grams_per_bird"`
}

// UpdateFoodAgeRulesRequest is the request payload for saving the whole feeding table.
type UpdateFoodAgeRulesRequest struct {
	Rules []FoodAgeRuleInput `json:"rules"`
}

// FoodRules is a user's whole feeding table, as loaded from the database.
type FoodRules []FoodAgeRule

// Match returns the rule whose age range contains ageWeeks, or nil if none does.
func (rs FoodRules) Match(ageWeeks int) *FoodAgeRule {
	for i := range rs {
		r := &rs[i]
		if ageWeeks >= r.MinWeeks && (r.MaxWeeks == 0 || ageWeeks < r.MaxWeeks) {
			return r
		}
	}
	return nil
}

// CoopFeed is the single calculation behind every food number: from a coop's age in weeks
// and its headcount it finds the owner's matching rule and returns the food type, the kg
// the whole coop eats per day (headcount x grams per bird / 1000), and the rule itself
// (so callers can explain why). Returns ("", 0, nil) when no rule matches.
func (rs FoodRules) CoopFeed(ageWeeks, headcount int) (foodType string, kgPerDay float64, rule *FoodAgeRule) {
	rule = rs.Match(ageWeeks)
	if rule == nil {
		return "", 0, nil
	}
	if headcount <= 0 {
		return rule.FoodType, 0, rule
	}
	return rule.FoodType, float64(headcount) * rule.GramsPerBird / 1000, rule
}

// Validate checks every row (known food type, ages >= 0, max 0 or > min, grams > 0) and
// that no two rows' age ranges overlap - an age must map to exactly one rule.
// Returns "" when valid.
func (r UpdateFoodAgeRulesRequest) Validate() string {
	for i, in := range r.Rules {
		n := i + 1
		if !IsValidFoodType(in.FoodType) {
			return "แถวที่ " + itoa(n) + ": ประเภทอาหารไม่ถูกต้อง"
		}
		if in.MinWeeks < 0 || in.MaxWeeks < 0 {
			return "แถวที่ " + itoa(n) + ": อายุต้องไม่ติดลบ"
		}
		if in.MaxWeeks != 0 && in.MaxWeeks <= in.MinWeeks {
			return "แถวที่ " + itoa(n) + ": อายุสิ้นสุดต้องมากกว่าอายุเริ่มต้น (หรือใส่ 0 = ไม่จำกัด)"
		}
		if in.GramsPerBird <= 0 {
			return "แถวที่ " + itoa(n) + ": ปริมาณอาหารต่อตัวต่อวัน (กรัม) ต้องมากกว่า 0"
		}
	}

	sorted := append([]FoodAgeRuleInput(nil), r.Rules...)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a].MinWeeks < sorted[b].MinWeeks })
	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1]
		if prev.MaxWeeks == 0 || prev.MaxWeeks > sorted[i].MinWeeks {
			return "ช่วงอายุของแต่ละแถวต้องไม่ซ้อนทับกัน"
		}
	}
	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
