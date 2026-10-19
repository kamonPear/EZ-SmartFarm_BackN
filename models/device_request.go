// =====================================================================================
// ไฟล์ models/device_request.go - รูปแบบ request/response ของอุปกรณ์ (รุ่นเก่า)
// =====================================================================================
//
// ตอนนี้ไม่มีที่ไหนใช้ struct ในไฟล์นี้ API อุปกรณ์รับ/ส่ง models.Device ตรง ๆ
package models

import "time"

// CreateDeviceRequest รูปแบบเพิ่มอุปกรณ์ (ไม่ได้ใช้แล้ว)
type CreateDeviceRequest struct {
	CoopID        int    `json:"coop_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	DeviceType    string `json:"device_type" binding:"required"`
	CurrentStatus string `json:"current_status"`
}

// UpdateDeviceRequest รูปแบบแก้ไขอุปกรณ์ (ไม่ได้ใช้แล้ว)
type UpdateDeviceRequest struct {
	Name          string `json:"name"`
	DeviceType    string `json:"device_type"`
	CurrentStatus string `json:"current_status"`
}

// DeviceResponse รูปแบบตอบกลับอุปกรณ์ (ไม่ได้ใช้แล้ว ระวังสับสนกับ handlers.DeviceResponse ซึ่งเป็นคนละตัว)
type DeviceResponse struct {
	DeviceID      int       `json:"device_id"`
	CoopID        int       `json:"coop_id"`
	Name          string    `json:"name"`
	DeviceType    string    `json:"device_type"`
	CurrentStatus string    `json:"current_status"`
	LastUpdate    time.Time `json:"last_update"`
}
