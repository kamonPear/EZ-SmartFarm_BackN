// =====================================================================================
// ไฟล์ models/importfood_request.go - รูปแบบ JSON ตอนนำเข้าอาหาร
// =====================================================================================
package models

import "time"

// CreateImportFoodRequest body ของ POST /api/importfoods: ประเภท, ปริมาณ, วันหมดอายุ
// บันทึกแล้วจะบวก ImportVolume เข้ายอดคงเหลือของประเภทนั้นด้วย (วันที่นำเข้า backend ใส่ให้)
type CreateImportFoodRequest struct {
	FoodType     string    `json:"food_type" binding:"required"`
	ImportVolume int       `json:"import_volume" binding:"required,min=0"`
	ExpiryDate   time.Time `json:"expiry_date" binding:"required"`
	// MinQuantity (optional) sets the owner's low-stock alert threshold (kg) for FoodType
	// at the same time; omitted/null leaves the existing threshold untouched.
	MinQuantity *float64 `json:"min_quantity,omitempty"`
}

// ImportFoodResponse รูปแบบตอบกลับแบบย่อ ตอนนี้ไม่มีที่ไหนใช้ (API ส่ง ImportFood ตรง ๆ)
type ImportFoodResponse struct {
	LotID        int       `json:"lot_id"`
	FoodType     string    `json:"food_type"`
	ImportVolume int       `json:"import_volume"`
	ImportDate   time.Time `json:"import_date"`
	ExpiryDate   time.Time `json:"expiry_date"`
}
