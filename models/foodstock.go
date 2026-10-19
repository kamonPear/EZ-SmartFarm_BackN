// =====================================================================================
// ไฟล์ models/foodstock.go - ประเภทอาหาร และโครงสร้างตาราง foodstock (ยอดอาหารคงเหลือ)
// =====================================================================================
package models

import "time"

// ประเภทอาหารที่ระบบรู้จัก (มีแค่ 2 แบบ) ทุกแถวของ foodstock / importfood ต้องใช้ค่าใดค่าหนึ่งนี้
// เขียนเป็นค่าคงที่ เพื่อไม่ต้องพิมพ์ข้อความไทยซ้ำหลายที่ (พิมพ์ผิดตัวเดียวจะกลายเป็นคนละประเภท)
const (
	FoodTypeSmallPellet = "เม็ดเล็ก" // ไก่อายุ 0-14 สัปดาห์
	FoodTypeLargePellet = "เม็ดใหญ่" // ไก่อายุตั้งแต่ 14 สัปดาห์ขึ้นไป
)

// IsValidFoodType ตรวจว่าเป็น "เม็ดเล็ก" หรือ "เม็ดใหญ่" หรือไม่
// ถูกเรียกจาก: handlers ที่รับประเภทอาหาร (importfood, food_distribution, foodstock) และ Validate ของตารางให้อาหาร
func IsValidFoodType(foodType string) bool {
	return foodType == FoodTypeSmallPellet || foodType == FoodTypeLargePellet
}

// Foodstock ยอดอาหารคงเหลือในคลัง 1 แถวต่อ (ผู้ใช้, ประเภทอาหาร) (unique uq_foodstock_type_user)
//  - QuantityCurrent : กก. ที่เหลือ มี check >= 0 กันยอดติดลบที่ระดับฐานข้อมูล
//  - DateUp          : วันที่ยอดเปลี่ยนล่าสุด
// ยอดเพิ่มเมื่อนำเข้าอาหาร ลดเมื่อตัดสต็อก
type Foodstock struct {
	FoodID          int       `gorm:"primaryKey;autoIncrement;column:food_id;type:int" json:"food_id"`
	UserID          int       `gorm:"column:user_id;type:int;size:32;uniqueIndex:uq_foodstock_type_user" json:"user_id"`
	FoodType        string    `gorm:"column:food_type;type:varchar(20);uniqueIndex:uq_foodstock_type_user" json:"food_type"`
	QuantityCurrent float64   `gorm:"column:quantity_current;type:decimal(10,2);check:quantity_current >= 0" json:"quantity_current"`
	DateUp          time.Time `gorm:"column:date_up;type:date;not null" json:"date_up"`
	// MinQuantity คือปริมาณที่เจ้าของฟาร์มกำหนดเอง - สต็อกเหลือน้อยกว่าหรือเท่านี้ (กก.)
	// แอปจะแจ้งเตือน "ใกล้หมด" 0 = ไม่ได้ตั้ง (ไม่แจ้งเตือนใกล้หมด แจ้งเฉพาะตอนหมด)
	MinQuantity float64 `gorm:"column:min_quantity;type:decimal(10,2);not null;default:0" json:"min_quantity"`
}

// TableName บอก GORM ว่าใช้ตาราง "foodstock"
func (Foodstock) TableName() string {
	return "foodstock"
}
