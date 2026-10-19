// =====================================================================================
// ไฟล์ models/vaccine_request.go - รูปแบบ JSON ตอนบันทึก/แก้ไขประวัติการให้วัคซีน
// =====================================================================================
package models

import "time"

// CreateVaccineRequest body ของ POST /api/vaccines: คอก, ชื่อวัคซีน, วิธีให้, อายุที่แนะนำ, วันที่ให้, หมายเหตุ
type CreateVaccineRequest struct {
	CoopID         int       `json:"coop_id" binding:"required"`
	Name           string    `json:"name" binding:"required"`
	Method         string    `json:"method" binding:"required"`
	RecommendedAge string    `json:"recommended_age"`
	RecordDate     time.Time `json:"record_date" binding:"required"`
	Note           string    `json:"note"`
}

// UpdateVaccineRequest body ของ PUT /api/vaccines?id= ช่องที่ว่าง/ศูนย์จะคงค่าเดิม
type UpdateVaccineRequest struct {
	Name           string    `json:"name"`
	Method         string    `json:"method"`
	RecommendedAge string    `json:"recommended_age"`
	RecordDate     time.Time `json:"record_date"`
	Note           string    `json:"note"`
}
