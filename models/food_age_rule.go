// =====================================================================================
// ไฟล์ models/food_age_rule.go - ตารางให้อาหารตามอายุไก่ และ "สูตรคำนวณอาหาร" ของทั้งระบบ
// =====================================================================================
//
// ทุกตัวเลขเรื่องอาหาร (ประมาณการรายคอก, ตัดสต็อกอัตโนมัติ, ตัดสต็อกเอง) มาจาก FoodRules.CoopFeed ตัวเดียว
// ตัวเลขจึงตรงกันทุกหน้าจอ
package models

import "sort"

// FoodAgeRule 1 แถวของตารางให้อาหารที่เจ้าของฟาร์มกรอกเอง:
// "ไก่อายุ [MinWeeks, MaxWeeks) สัปดาห์ กินอาหาร FoodType ตัวละ GramsPerBird กรัมต่อวัน"
//  - MaxWeeks = 0 หมายถึงไม่จำกัดอายุสูงสุด
//  - เพิ่มได้หลายแถว (หลายช่วงอายุต่อประเภทอาหาร)
// ไม่มีค่าตายตัวในโค้ด: คอกที่อายุไม่ตรงแถวไหนเลย = ไม่มีประเภทอาหาร ไม่ถูกตัดสต็อก
type FoodAgeRule struct {
	ID           int     `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID       int     `gorm:"column:user_id;type:int;size:32;index:idx_food_age_rule_user" json:"user_id"`
	FoodType     string  `gorm:"column:food_type;type:varchar(20);not null" json:"food_type"`
	MinWeeks     int     `gorm:"column:min_weeks;type:int;not null" json:"min_weeks"`
	MaxWeeks     int     `gorm:"column:max_weeks;type:int;not null;default:0" json:"max_weeks"`
	GramsPerBird float64 `gorm:"column:grams_per_bird;type:decimal(10,3);not null" json:"grams_per_bird"`
}

// TableName บอก GORM ว่าใช้ตาราง "food_age_rule"
func (FoodAgeRule) TableName() string {
	return "food_age_rule"
}

// FoodAgeRuleInput 1 แถวใน body ตอนบันทึกตาราง (ไม่มี id/user_id เพราะบันทึกแทนที่ทั้งตาราง)
type FoodAgeRuleInput struct {
	FoodType     string  `json:"food_type"`
	MinWeeks     int     `json:"min_weeks"`
	MaxWeeks     int     `json:"max_weeks"`
	GramsPerBird float64 `json:"grams_per_bird"`
}

// UpdateFoodAgeRulesRequest body ของ PUT /api/food-age-rules = ทั้งตาราง {"rules": [...]}
type UpdateFoodAgeRulesRequest struct {
	Rules []FoodAgeRuleInput `json:"rules"`
}

// FoodRules ตารางให้อาหารทั้งหมดของผู้ใช้ 1 คน (slice ของ FoodAgeRule)
// ตั้งเป็น type ของตัวเอง เพื่อผูก method Match และ CoopFeed ไว้ใช้ได้เลย
type FoodRules []FoodAgeRule

// Match หาแถวที่ช่วงอายุครอบ ageWeeks (MinWeeks <= อายุ < MaxWeeks หรือ MaxWeeks = 0)
// ไม่เจอคืน nil คืน pointer ไปที่แถวใน slice (ไม่คัดลอก)
// ถูกเรียกจาก: CoopFeed
func (rs FoodRules) Match(ageWeeks int) *FoodAgeRule {
	for i := range rs {
		r := &rs[i]
		if ageWeeks >= r.MinWeeks && (r.MaxWeeks == 0 || ageWeeks < r.MaxWeeks) {
			return r
		}
	}
	return nil
}

// CoopFeed "สูตรเดียว" ของทุกตัวเลขอาหารในระบบ
// รับอายุคอก (สัปดาห์) และจำนวนไก่ แล้วคืน:
//  - foodType : ประเภทอาหารที่คอกนี้ควรกิน
//  - kgPerDay : กก./วัน ทั้งคอก = จำนวนไก่ x กรัมต่อตัว / 1000
//  - rule     : แถวที่ใช้ (ให้ผู้เรียกอธิบายได้ว่าทำไมได้ค่านี้)
// ไม่มีแถวตรง -> ("", 0, nil), จำนวนไก่ <= 0 -> ได้ประเภทแต่ kg = 0
// ถูกเรียกจาก: handlers/foodstock.go (ComputeDailyFoodConsumptionByCoop) และ handlers/coop_food_consumption.go
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

// Validate ตรวจตารางให้อาหารก่อนบันทึก คืน "" ถ้าผ่าน หรือข้อความภาษาไทยบอกว่าแถวไหนผิด
// ตรวจทีละแถว: ประเภทอาหารถูกต้อง, อายุไม่ติดลบ, อายุสิ้นสุด > อายุเริ่ม (หรือ 0), กรัม > 0
// แล้วเรียงตามอายุเริ่ม เช็กว่าช่วงอายุไม่ซ้อนกัน
// ทำไมห้ามซ้อน: อายุหนึ่งต้องได้แถวเดียว ไม่งั้นไม่รู้จะใช้อาหารแบบไหน
// ถูกเรียกจาก: handlers/food_age_rule.go
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

// itoa แปลงเลขจำนวนเต็มบวกเป็นข้อความ (เช่น 3 -> "3") ใช้ทำข้อความ "แถวที่ N" ใน Validate
// (ทำงานเหมือน strconv.Itoa สำหรับเลขบวก)
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
