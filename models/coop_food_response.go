// =====================================================================================
// ไฟล์ models/coop_food_response.go - รูปแบบผลลัพธ์ "อาหารที่แต่ละคอกกินต่อวัน"
// =====================================================================================
package models

// CoopFoodConsumption ผลการประมาณอาหารของ 1 คอก ส่งจาก GET /api/foods/coop-consumption
//  - AgeWeeks / FoodType / EstimatedKgPerDay : อายุไก่ (สัปดาห์), ประเภทอาหาร, กก./วัน
//  - AgeMinWeeks / AgeMaxWeeks / GramsPerBird : ช่วงอายุในตารางให้อาหารที่คอกนี้ตกอยู่ และกรัม/ตัว/วัน
//    ใช้บอกผู้ใช้ "ทำไม" คอกนี้ได้ค่านี้ (max 0 = ไม่จำกัดอายุสูงสุด)
// เป็นค่าประมาณจากจำนวนไก่ (ไม่มีเซนเซอร์วัดอาหาร) แต่สูตรเดียวกับที่ใช้ตัดสต็อกจริง
// ไม่ใช่ตาราง ใช้แค่ส่งออก JSON
type CoopFoodConsumption struct {
	CoopID            int     `json:"coop_id"`
	NameCoop          string  `json:"name_coop"`
	Amount            int     `json:"amount"`
	AgeWeeks          int     `json:"age_weeks"`
	FoodType          string  `json:"food_type"`
	EstimatedKgPerDay float64 `json:"estimated_kg_per_day"`
	// Why this coop is on FoodType: the owner's configured range (weeks, max 0 = no limit)
	// that its AgeWeeks fell into, and the grams one bird of that range eats per day.
	AgeMinWeeks  int     `json:"age_min_weeks"`
	AgeMaxWeeks  int     `json:"age_max_weeks"`
	GramsPerBird float64 `json:"grams_per_bird"`
}
