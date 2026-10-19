// =====================================================================================
// ไฟล์ models/egg_request.go - รูปแบบ JSON ตอนเพิ่ม/แก้ไขบันทึกไข่
// =====================================================================================
package models

import "time"

// CreateEggRequest body ของ POST /api/eggs: คอก, วันที่เก็บ, จำนวนไข่, หมายเหตุ
type CreateEggRequest struct {
	CoopID         int       `json:"coop_id" binding:"required"`
	DateCollectEgg time.Time `json:"date_collect_egg" binding:"required"`
	NumberEgg      int       `json:"number_egg" binding:"required,min=1"`
	Note           string    `json:"note"`
}

// UpdateEggRequest body ของ PUT /api/eggs?id= ช่องที่เป็นค่าว่าง/ศูนย์ไม่ถูกแก้
type UpdateEggRequest struct {
	CoopID         int       `json:"coop_id"`
	DateCollectEgg time.Time `json:"date_collect_egg"`
	NumberEgg      int       `json:"number_egg" binding:"min=1"`
	Note           string    `json:"note"`
}

// EggResponse รูปแบบตอบกลับแบบย่อ ตอนนี้ไม่มีที่ไหนใช้ (API ส่ง Egg ตรง ๆ)
type EggResponse struct {
	EggID          int       `json:"egg_id"`
	CoopID         int       `json:"coop_id"`
	DateCollectEgg time.Time `json:"date_collect_egg"`
	NumberEgg      int       `json:"number_egg"`
	Note           string    `json:"note"`
}
