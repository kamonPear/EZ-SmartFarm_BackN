// =====================================================================================
// ไฟล์ models/farm_layout.go - โครงสร้างตาราง farm_layout (รูปทรงขอบฟาร์มบนหน้าผังฟาร์ม)
// =====================================================================================
package models

// รูปทรงขอบฟาร์มที่หน้าเว็บวาดได้ มีแค่ 3 แบบ: วงกลม สามเหลี่ยม สี่เหลี่ยม
const (
	FarmShapeCircle   = "circle"
	FarmShapeTriangle = "triangle"
	FarmShapeSquare   = "square"
)

// IsValidFarmShape ตรวจว่ารูปทรงที่ส่งมาเป็น 1 ใน 3 แบบที่รองรับหรือไม่
// ถูกเรียกจาก: handlers/farm_layout.go ก่อนบันทึก (กันค่าที่หน้าเว็บวาดไม่ได้)
func IsValidFarmShape(shape string) bool {
	return shape == FarmShapeCircle || shape == FarmShapeTriangle || shape == FarmShapeSquare
}

// FarmLayout รูปทรงขอบฟาร์มที่ผู้ใช้เลือก 1 แถวต่อผู้ใช้ (unique user_id)
// รุ่นเก่าใช้แถวเดียว (id=1) ร่วมกันทุกคน ตอนนี้แยกต่อผู้ใช้ และสร้างแถวให้ตอนอ่าน/เขียนครั้งแรก
type FarmLayout struct {
	ID     int    `gorm:"primaryKey;autoIncrement;column:id;type:int" json:"id"`
	UserID int    `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_farm_layout_user" json:"user_id"`
	Shape  string `gorm:"column:shape;type:varchar(20)" json:"shape"`
}

// TableName บอก GORM ว่าใช้ตาราง "farm_layout"
func (FarmLayout) TableName() string {
	return "farm_layout"
}
