package models

// DeviceType คือ "ชนิด" อุปกรณ์ที่ใช้ซ้ำตอนจัดวางอุปกรณ์ลงคอก (เช่นซื้อเซนเซอร์รุ่น
// ใหม่มา ก็มาเพิ่มชื่อ+ไอคอนไว้ที่นี่ครั้งเดียว แล้วเลือกใช้ได้ทุกคอกในฟาร์มตัวเอง)
// แยกเป็นต่างหากจากตาราง Device ซึ่งเป็นอุปกรณ์ที่ถูกวางจริงในคอกใดคอกหนึ่งแล้ว -
// user ใหม่เริ่มจากไม่มีชนิดอุปกรณ์เลย ต้องเพิ่มเอง (ไม่มีการ seed ให้อัตโนมัติ)
type DeviceType struct {
	ID     int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID int    `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	Name   string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// text ไม่ใช่ varchar(255) - ไอคอนเก็บเป็น SVG data: URI ที่เข้ารหัสแล้วยาว
	// 500+ ตัวอักษร ของเดิม varchar(255) ทำให้ INSERT พัง (MySQL strict mode ปฏิเสธ
	// แถวที่ยาวเกินคอลัมน์)
	Icon string `gorm:"column:icon;type:text;not null" json:"icon"`
}

func (DeviceType) TableName() string {
	return "device_type"
}
