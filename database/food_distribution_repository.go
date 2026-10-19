// =====================================================================================
// ไฟล์ database/food_distribution_repository.go - ประวัติ "การแจกจ่ายอาหาร" ให้แต่ละคอก
// =====================================================================================
//
// ทุกครั้งที่ตัดสต็อกอาหาร (อัตโนมัติ 18:00 น. หรือกด "ตัดสต็อก" เอง) ระบบจะบันทึกว่า
// แต่ละคอกได้อาหารประเภทไหนไปกี่กิโลกรัม เก็บในตาราง food_distributions
// ใช้แสดง "ประวัติการนำออก/แจกจ่ายอาหาร" ในหน้าคลังอาหาร และใช้เช็กว่าวันนี้ตัดสต็อกไปแล้วหรือยัง
package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"
)

// CreateFoodDistributionBatch บันทึกการแจกจ่ายอาหาร 1 ครั้ง (1 แถวต่อ 1 คอก)
//
// รับค่า: foodType (เม็ดเล็ก/เม็ดใหญ่), items (คอกไหนได้กี่ กก.), userID
// ทุกแถวในรอบเดียวกันใช้เวลา DistributedAt เดียวกัน (now) เพื่อจัดกลุ่มกลับเป็น "รอบเดียวกัน" ได้
// DB.Create(&rows) แบบส่ง slice = insert ทีเดียวหลายแถว เร็วกว่าวนทีละแถว
// ถูกเรียกจาก: handlers/foodstock.go (ตัดสต็อกอัตโนมัติ/ตัดเอง) และ handlers/food_distribution.go
func CreateFoodDistributionBatch(foodType string, items []models.FoodDistributionItem, userID int) ([]models.FoodDistribution, error) {
	now := time.Now()
	rows := make([]models.FoodDistribution, 0, len(items))
	for _, item := range items {
		rows = append(rows, models.FoodDistribution{
			UserID:        userID,
			FoodType:      foodType,
			CoopID:        item.CoopID,
			KgGiven:       item.KgGiven,
			DistributedAt: now,
		})
	}

	if err := DB.Create(&rows).Error; err != nil {
		log.Printf("Error creating food distribution batch: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Recorded food distribution: %s across %d coops (user %d)\n", foodType, len(rows), userID)
	return rows, nil
}

// GetFoodDistributionHistory ดึงประวัติแจกจ่ายอาหารทั้งหมดของผู้ใช้ เรียงจากใหม่ไปเก่า
// ถ้าส่ง foodType มา จะกรองเฉพาะประเภทนั้น
// Preload("Coop") = ดึงข้อมูลคอกที่ผูกอยู่มาด้วยในคราวเดียว จะได้แสดงชื่อคอกได้เลย
// ถูกเรียกจาก: handlers/food_distribution.go (GET /api/foods/distribution)
func GetFoodDistributionHistory(foodType string, userID int) ([]models.FoodDistribution, error) {
	var rows []models.FoodDistribution
	query := DB.Preload("Coop").Where("user_id = ?", userID).Order("distributed_at DESC")
	if foodType != "" {
		query = query.Where("food_type = ?", foodType)
	}
	if err := query.Find(&rows).Error; err != nil {
		log.Printf("Error fetching food distribution history: %v", err)
		return nil, err
	}
	return rows, nil
}

// GetLatestDistributionDate หาวันเวลาล่าสุดที่เคยตัดสต็อกอาหารประเภทนี้ของผู้ใช้
// คืนค่า (เวลา, เจอไหม, error) ถ้าไม่เคยตัดเลยคืน found = false
//
// ใช้ทำอะไร: handlers/foodstock.go (ตัวตัดสต็อกตามหลัง / catch-up) เช็กว่า "วันนี้ตัดไปแล้วหรือยัง"
// กันการตัดซ้ำสองครั้งในวันเดียว กรณีเซิร์ฟเวอร์ Render หลับไปตอน 18:00 แล้วตื่นทีหลัง
func GetLatestDistributionDate(userID int, foodType string) (time.Time, bool, error) {
	var row models.FoodDistribution
	err := DB.Where("user_id = ? AND food_type = ?", userID, foodType).
		Order("distributed_at DESC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return row.DistributedAt, true, nil
}

// DeleteAllFoodDistribution ลบประวัติแจกจ่ายอาหาร "ทั้งหมดของผู้ใช้คนนั้น"
// ต้องกรอง user_id เสมอ ถ้าลบทั้งตาราง ผู้ใช้คนหนึ่งจะลบข้อมูลของทุกฟาร์มได้
// ถูกเรียกจาก: handlers/food_distribution.go (DELETE /api/foods/distribution)
func DeleteAllFoodDistribution(userID int) error {
	// Scoped to the caller - "delete all" means all of *their* history. A global
	// wipe here would let any logged-in user destroy every other user's records.
	if err := DB.Where("user_id = ?", userID).Delete(&models.FoodDistribution{}).Error; err != nil {
		log.Printf("Error deleting food distribution history: %v", err)
		return err
	}
	log.Println("✓ Food distribution history cleared")
	return nil
}
