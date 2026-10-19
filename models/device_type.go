// =====================================================================================
// ไฟล์ models/device_type.go - โครงสร้างตาราง device_type (ชนิดอุปกรณ์ที่ผู้ใช้เพิ่มเอง)
// =====================================================================================
package models

// DeviceType "แม่แบบ" อุปกรณ์ (ชื่อ + ไอคอน) เพิ่มครั้งเดียว ใช้วางได้ทุกคอกในฟาร์มตัวเอง
// เช่น ซื้อเซนเซอร์รุ่นใหม่มา ก็เพิ่มชนิดไว้ที่นี่ แล้วลากไปวางในคอกต่าง ๆ
// แยกจากตาราง device (อุปกรณ์ที่วางจริงในคอกแล้ว) ผู้ใช้ใหม่เริ่มจากรายการว่าง ต้องเพิ่มเอง
// Icon เป็น text เพราะไอคอน SVG ยาวเกิน 255 ตัวอักษร
type DeviceType struct {
	ID     int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID int    `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	Name   string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// text ไม่ใช่ varchar(255) - ไอคอนเก็บเป็น SVG data: URI ที่เข้ารหัสแล้วยาว
	// 500+ ตัวอักษร ของเดิม varchar(255) ทำให้ INSERT พัง (MySQL strict mode ปฏิเสธ
	// แถวที่ยาวเกินคอลัมน์)
	Icon string `gorm:"column:icon;type:text;not null" json:"icon"`
}

// TableName บอก GORM ว่าใช้ตาราง "device_type"
func (DeviceType) TableName() string {
	return "device_type"
}
