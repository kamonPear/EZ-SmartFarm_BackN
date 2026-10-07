package models

// DeviceType คือ "ชนิด" อุปกรณ์ที่ใช้ซ้ำตอนจัดวางอุปกรณ์ลงคอก (เช่นซื้อเซนเซอร์รุ่น
// ใหม่มา ก็มาเพิ่มชื่อ+ไอคอนไว้ที่นี่ครั้งเดียว แล้วเลือกใช้ได้ทุกคอกในฟาร์มตัวเอง)
// แยกเป็นต่างหากจากตาราง Device ซึ่งเป็นอุปกรณ์ที่ถูกวางจริงในคอกใดคอกหนึ่งแล้ว -
// ชนิดอุปกรณ์พื้นฐาน 7 แบบ (ESP32/MQ-135/ฯลฯ) ก็เป็นแถวจริงในตารางนี้เหมือนกัน
// (seed ให้ user ทุกคนอัตโนมัติ ดู handlers.ensureDefaultDeviceTypes) แก้ไข/ลบได้
// เหมือนชนิดที่ผู้ใช้เพิ่มเองทุกประการ ไม่ใช่ค่าคงที่ฝัง frontend อีกต่อไป
type DeviceType struct {
	ID     int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID int    `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	Name   string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// text ไม่ใช่ varchar(255) - ไอคอนเก็บเป็น SVG data: URI ที่เข้ารหัสแล้วยาว
	// 500+ ตัวอักษร ของเดิม varchar(255) ทำให้ INSERT พัง (MySQL strict mode ปฏิเสธ
	// แถวที่ยาวเกินคอลัมน์) seed ชนิดมาตรฐาน 7 แบบไม่ติดเลยสักครั้งเพราะบั๊กนี้
	Icon string `gorm:"column:icon;type:text;not null" json:"icon"`
}

func (DeviceType) TableName() string {
	return "device_type"
}
