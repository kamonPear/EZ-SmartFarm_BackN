// =====================================================================================
// ไฟล์ models/health_appointment_request.go - รูปแบบ JSON ตอนสร้างนัดตรวจสุขภาพ
// =====================================================================================
package models

import "time"

// CreateHealthAppointmentRequest body ของ POST /api/health-appointments
// วันที่ส่งเป็นข้อความรูปแบบ RFC3339 เช่น "2026-10-05T00:00:00Z" Go แปลงเป็น time.Time ให้อัตโนมัติ
// (รูปแบบเดียวกับวันที่ของไข่/สุขภาพ/วัคซีน)
type CreateHealthAppointmentRequest struct {
	CoopID          int       `json:"coop_id"`
	AppointmentDate time.Time `json:"appointment_date"`
}
