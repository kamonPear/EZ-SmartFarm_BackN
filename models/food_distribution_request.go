// =====================================================================================
// ไฟล์ models/food_distribution_request.go - รูปแบบ JSON ตอนบันทึกการแจกอาหาร
// =====================================================================================
package models

// FoodDistributionItem ส่วนของ 1 คอก: coop_id และ kg ที่ให้
// ใช้ทั้งใน request และเป็นผลคำนวณของ ComputeDailyFoodConsumptionByCoop
type FoodDistributionItem struct {
	CoopID  int     `json:"coop_id"`
	KgGiven float64 `json:"kg_given"`
}

// RecordFoodDistributionRequest body ของ POST /api/foods/distribution
// ประเภทอาหาร + รายการคอก ยอดที่ตัดสต็อก = ผลรวม KgGiven ทุกคอกพอดี
type RecordFoodDistributionRequest struct {
	FoodType string                 `json:"food_type"`
	Items    []FoodDistributionItem `json:"items"`
}
