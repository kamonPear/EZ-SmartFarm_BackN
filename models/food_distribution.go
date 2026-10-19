// =====================================================================================
// ไฟล์ models/food_distribution.go - โครงสร้างตาราง food_distribution (ประวัติอาหารที่แต่ละคอกได้รับ)
// =====================================================================================
package models

import "time"

// FoodDistribution 1 แถว = คอกหนึ่งได้อาหารไปกี่ กก. ในการตัดสต็อกครั้งหนึ่ง
// เก็บเป็นประวัติถาวร (ไม่เขียนทับ) สร้างทั้งตอนผู้ใช้กรอกเอง และตอนระบบตัดสต็อกอัตโนมัติ
// KgGiven เป็นทศนิยม 4 ตำแหน่ง เพราะลูกไก่กินวันละไม่ถึง 0.001 กก.
// ยังใช้เช็ก "วันนี้ตัดสต็อกไปแล้วหรือยัง" ใน RunCatchUpDeduction ด้วย
type FoodDistribution struct {
	DistributionID int       `gorm:"primaryKey;autoIncrement;column:distribution_id;type:int" json:"distribution_id"`
	UserID         int       `gorm:"column:user_id;type:int;size:32;index" json:"user_id"`
	FoodType       string    `gorm:"column:food_type;type:varchar(20);not null;index" json:"food_type"`
	CoopID         int       `gorm:"column:coop_id;not null;type:int" json:"coop_id"`
	KgGiven        float64   `gorm:"column:kg_given;type:decimal(10,4);not null" json:"kg_given"`
	DistributedAt  time.Time `gorm:"column:distributed_at;not null" json:"distributed_at"`

	Coop Coop `gorm:"foreignKey:CoopID;constraint:-" json:"coop,omitempty"`
}

// TableName บอก GORM ว่าใช้ตาราง "food_distribution"
func (FoodDistribution) TableName() string {
	return "food_distribution"
}
