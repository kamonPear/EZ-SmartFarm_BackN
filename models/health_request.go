// =====================================================================================
// ไฟล์ models/health_request.go - รูปแบบ JSON ตอนเพิ่ม/แก้ไขผลตรวจสุขภาพ
// =====================================================================================
package models

import "time"

// CreateHealthRequest body ของ POST /api/healths: คอก, จำนวนดี, จำนวนป่วย, วันที่, หมายเหตุ
type CreateHealthRequest struct {
	CoopID     int       `json:"coop_id" binding:"required"`
	Healthy    int       `json:"healthy" binding:"required"`
	PoorHealth int       `json:"poor_health" binding:"required"`
	RecordDate time.Time `json:"record_date" binding:"required"`
	Note       string    `json:"note"`
}

// UpdateHealthRequest body ของ PUT /api/healths?id= (แก้เฉพาะช่องที่ส่งมา)
type UpdateHealthRequest struct {
	CoopID     int       `json:"coop_id"`
	Healthy    int       `json:"healthy"`
	PoorHealth int       `json:"poor_health"`
	RecordDate time.Time `json:"record_date"`
	Note       string    `json:"note"`
}

// HealthResponse รูปแบบตอบกลับแบบย่อ ตอนนี้ไม่มีที่ไหนใช้ (API ส่ง Health ตรง ๆ)
type HealthResponse struct {
	HealthID   int       `json:"health_id"`
	CoopID     int       `json:"coop_id"`
	Healthy    int       `json:"healthy"`
	PoorHealth int       `json:"poor_health"`
	RecordDate time.Time `json:"record_date"`
	Note       string    `json:"note"`
}
