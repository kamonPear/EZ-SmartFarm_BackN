// =====================================================================================
// ไฟล์ models/foodstock_request.go - รูปแบบ JSON ของคลังอาหาร
// =====================================================================================
package models

import "time"

// UpdateFoodstockRequest body ของ PUT /api/foods?id= แก้ยอดคงเหลือให้ตรงของจริง (และวันที่ ถ้าส่งมา)
type UpdateFoodstockRequest struct {
	QuantityCurrent float64   `json:"quantity_current" binding:"min=0"`
	DateUp          time.Time `json:"date_up"`
}

// UpdateFoodMinQuantityRequest body ของ PUT /api/foods/min-quantity: ปริมาณใกล้หมด (กก.)
// (food_type ยังรับไว้เพื่อให้แอปรุ่นเก่าส่งมาได้ แต่ค่าใช้ร่วมกันทุกประเภท)
type UpdateFoodMinQuantityRequest struct {
	FoodType    string  `json:"food_type"`
	MinQuantity float64 `json:"min_quantity"`
}

// FoodstockResponse รูปแบบตอบกลับแบบย่อ ตอนนี้ไม่มีที่ไหนใช้ (API ส่ง Foodstock ตรง ๆ)
type FoodstockResponse struct {
	FoodID          int       `json:"food_id"`
	FoodType        string    `json:"food_type"`
	QuantityCurrent float64   `json:"quantity_current"`
	DateUp          time.Time `json:"date_up"`
}

// DeductFoodstockRequest body ของ POST /api/foodstocks/force-deduct บอกแค่ประเภทอาหาร
// (ยอดที่ตัดระบบคำนวณเองจากจำนวนไก่)
type DeductFoodstockRequest struct {
	FoodType string `json:"food_type" binding:"required"`
}
