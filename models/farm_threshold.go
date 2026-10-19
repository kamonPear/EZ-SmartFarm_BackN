// =====================================================================================
// ไฟล์ models/farm_threshold.go - โครงสร้างตาราง farm_threshold (ค่ามาตรฐานอุณหภูมิ/แอมโมเนีย)
// =====================================================================================
package models

// FarmThreshold ค่าอุณหภูมิ (°C) และแอมโมเนีย (ppm) เป้าหมายของฟาร์ม ใช้ร่วมทุกคอก (ไม่ใช่รายคอก)
// 1 แถวต่อผู้ใช้ สร้างครั้งแรกเมื่อผู้ใช้ตั้งค่า (ยังไม่ตั้ง = ไม่มีแถว เห็นเป็น 0/0
// ไม่ใช่ค่าของผู้ใช้อื่นหรือค่าตั้งต้นที่ค้างอยู่)
// ตอนนี้เป็นค่าอ้างอิงที่แสดงบนหน้าจอ และส่งให้บอร์ดทาง MQTT
// ยังไม่ได้ผูกกับการเปิด/ปิดพัดลม/หลอดไฟอัตโนมัติ
type FarmThreshold struct {
	ID          int     `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID      int     `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_farm_threshold_user" json:"user_id"`
	Temperature float64 `gorm:"column:temperature;type:decimal(5,2);not null" json:"temperature"`
	Ammonia     float64 `gorm:"column:ammonia;type:decimal(5,2);not null" json:"ammonia"`
}

// TableName บอก GORM ว่าใช้ตาราง "farm_threshold"
func (FarmThreshold) TableName() string {
	return "farm_threshold"
}
