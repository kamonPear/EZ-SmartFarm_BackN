// =====================================================================================
// ไฟล์ models/farm_layout_request.go - รูปแบบ JSON ของหน้าจัดวางผังฟาร์ม
// =====================================================================================
package models

// UpdateFarmLayoutRequest body ของ PUT /api/farm-layout เช่น {"shape": "circle"}
type UpdateFarmLayoutRequest struct {
	Shape string `json:"shape" binding:"required"`
}

// CoopPositionEntry ตำแหน่งของ 1 คอกบนผัง (เปอร์เซ็นต์)
// PosX/PosY = nil คือเอาคอกออกจากผัง กลับไปอยู่ในถาด "ยังไม่ได้วาง"
type CoopPositionEntry struct {
	CoopID int      `json:"coop_id"`
	PosX   *float64 `json:"pos_x"`
	PosY   *float64 `json:"pos_y"`
}

// UpdateCoopPositionsRequest body ของ PUT /api/coops/positions = ตำแหน่งทุกคอกในครั้งเดียว
// (เหมือนหน้าจัดวางอุปกรณ์ที่บันทึกทุกช่องพร้อมกัน)
type UpdateCoopPositionsRequest struct {
	Positions []CoopPositionEntry `json:"positions"`
}

// UpdateCoopOrderRequest body ของ PUT /api/coops/order = รหัสคอกเรียงตามลำดับที่ผู้ใช้ลากจัดไว้
// ตัวแรกในรายการ = แสดงก่อนสุด
type UpdateCoopOrderRequest struct {
	CoopIDs []int `json:"coop_ids"`
}
