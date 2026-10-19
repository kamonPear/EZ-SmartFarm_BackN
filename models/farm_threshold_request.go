// =====================================================================================
// ไฟล์ models/farm_threshold_request.go - รูปแบบ JSON ตอนตั้งค่ามาตรฐาน
// =====================================================================================
package models

// UpdateFarmThresholdRequest body ของ PUT /api/farm-threshold เช่น {"temperature": 25, "ammonia": 35}
type UpdateFarmThresholdRequest struct {
	Temperature float64 `json:"temperature"`
	Ammonia     float64 `json:"ammonia"`
}
