package models

// DeviceType คือ "ชนิด" อุปกรณ์ที่ผู้ใช้เพิ่มเองไว้ใช้ซ้ำตอนจัดวางอุปกรณ์ลงคอก
// (เช่นซื้อเซนเซอร์รุ่นใหม่มา ก็มาเพิ่มชื่อ+ไอคอนไว้ที่นี่ครั้งเดียว แล้วเลือกใช้ได้
// ทุกคอกในฟาร์มตัวเอง) แยกเป็นต่างหากจากตาราง Device ซึ่งเป็นอุปกรณ์ที่ถูกวางจริง
// ในคอกใดคอกหนึ่งแล้ว - ชนิดอุปกรณ์พื้นฐาน 7 แบบ (ESP32/MQ-135/ฯลฯ) ยังคงเป็นค่า
// ตั้งต้นฝั่ง frontend ไม่ได้ย้ายมาเก็บที่นี่ด้วย ตารางนี้เก็บแค่ชนิดที่ผู้ใช้เพิ่มเอง
type DeviceType struct {
	ID     int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID int    `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	Name   string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Icon   string `gorm:"column:icon;type:varchar(255);not null" json:"icon"`
}

func (DeviceType) TableName() string {
	return "device_type"
}
