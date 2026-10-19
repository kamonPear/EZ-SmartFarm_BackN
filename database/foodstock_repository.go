// =====================================================================================
// ไฟล์ database/foodstock_repository.go - ยอดอาหารคงเหลือในคลัง (ตาราง foodstock)
// =====================================================================================
//
// ผู้ใช้ 1 คนมีได้ 1 แถวต่ออาหาร 1 ประเภท:
//   - เม็ดเล็ก : สำหรับไก่อายุ 0-14 สัปดาห์
//   - เม็ดใหญ่ : สำหรับไก่อายุ 14 สัปดาห์ขึ้นไป
//
// ยอดเพิ่มเมื่อนำเข้าล็อตใหม่ (importfood_repository.go -> addToFoodstock)
// ยอดลดเมื่อตัดสต็อกประจำวัน (DeductFoodstockByType) หรือผู้ใช้แก้ยอดเอง (UpdateFoodstock)
// ถูกเรียกจาก: handlers/foodstock.go และ handlers/food_distribution.go
package database

import (
	"errors"
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFoodstockByID ดึงแถวสต็อกตาม food_id เฉพาะของผู้ใช้คนนั้น ไม่เจอคืน ErrNotFound
// ถูกเรียกจาก: handlers/foodstock.go (GET /api/foods?id=) และ DeleteFoodstock ด้านล่าง
func GetFoodstockByID(foodID int, userID int) (*models.Foodstock, error) {
	var foodstock *models.Foodstock

	if err := DB.Where("food_id = ? AND user_id = ?", foodID, userID).
		First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error fetching foodstock: %v", err)
		return nil, err
	}

	return foodstock, nil
}

// GetAllFoodstocks ดึงสต็อกทุกประเภทของผู้ใช้ ใช้แสดง "สรุปสต็อกปัจจุบัน" ในหน้าคลังอาหาร
// ถูกเรียกจาก: handlers/foodstock.go
func GetAllFoodstocks(userID int) ([]models.Foodstock, error) {
	var foodstocks []models.Foodstock

	if err := DB.Where("user_id = ?", userID).Find(&foodstocks).Error; err != nil {
		log.Printf("Error fetching foodstocks: %v", err)
		return nil, err
	}

	return foodstocks, nil
}

// UpdateFoodstock แก้ยอดคงเหลือให้ตรงกับของจริงที่นับได้ (ส่วน "อัปเดตสต็อกอาหาร")
// ตั้ง QuantityCurrent เป็นค่าใหม่ทันที (ไม่ใช่บวก/ลบ) และอัปเดตวันที่ DateUp
// ถ้าไม่ส่งวันที่มา ใช้เวลาปัจจุบัน
// ถูกเรียกจาก: handlers/foodstock.go -> UpdateFoodstockHandler (PUT /api/foods)
func UpdateFoodstock(id int, req *models.UpdateFoodstockRequest, userID int) (*models.Foodstock, error) {
	var foodstock models.Foodstock
	if err := DB.Where("food_id = ? AND user_id = ?", id, userID).First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	foodstock.QuantityCurrent = req.QuantityCurrent
	if !req.DateUp.IsZero() {
		foodstock.DateUp = req.DateUp
	} else {
		foodstock.DateUp = time.Now()
	}

	if err := DB.Save(&foodstock).Error; err != nil {
		return nil, err
	}
	return &foodstock, nil
}

// ErrMinQuantityLocked error เมื่อเจ้าของฟาร์มเคยตั้ง "ปริมาณใกล้หมด" ไปแล้ว
// ตั้งได้ครั้งเดียว หลังจากนั้นล็อก แก้ไขหรือล้างค่าไม่ได้
var ErrMinQuantityLocked = errors.New("low-stock threshold is already set and locked")

// SetFoodMinQuantity ตั้ง "ปริมาณใกล้หมด" (กก.) ครั้งเดียว ถ้าสต็อกต่ำกว่านี้ระบบจะแจ้งเตือนว่าใกล้หมด
// เป็นค่าเดียวต่อเจ้าของฟาร์ม ใช้ร่วมกันทุกประเภทอาหาร จึงเขียนลงทุกแถวสต็อกของผู้ใช้
// (แถวที่สร้างทีหลังจะได้ค่านี้ตามไปด้วย ดู addToFoodstock)
//  - ต้องมีแถวสต็อกอย่างน้อย 1 แถว (สร้างตอนนำเข้าอาหารครั้งแรก) ไม่งั้นคืน ErrNotFound
//  - ถ้าเคยตั้งไปแล้ว ไม่แก้อะไร และคืน ErrMinQuantityLocked
// ถูกเรียกจาก: handlers/foodstock.go -> UpdateFoodMinQuantityHandler (PUT /api/foods/min-quantity)
func SetFoodMinQuantity(userID int, minQuantity float64) error {
	if getSharedMinQuantity(userID) > 0 {
		return ErrMinQuantityLocked
	}
	result := DB.Model(&models.Foodstock{}).Where("user_id = ?", userID).Update("min_quantity", minQuantity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// getSharedMinQuantity คืนปริมาณใกล้หมดที่ผู้ใช้เคยตั้งไว้ (0 = ยังไม่เคยตั้ง)
// ใช้เช็กว่าล็อกแล้วหรือยัง และให้อาหารประเภทที่นำเข้าครั้งแรกได้ค่านี้ตามไปด้วย
func getSharedMinQuantity(userID int) float64 {
	var row models.Foodstock
	if err := DB.Where("user_id = ? AND min_quantity > 0", userID).
		Order("min_quantity desc").First(&row).Error; err != nil {
		return 0
	}
	return row.MinQuantity
}

// DeleteFoodstock ลบแถวสต็อกหนึ่งแถว (เช็กเจ้าของผ่าน GetFoodstockByID ก่อน)
// ถูกเรียกจาก: handlers/foodstock.go (DELETE /api/foods)
func DeleteFoodstock(foodID int, userID int) error {
	foodstock, err := GetFoodstockByID(foodID, userID)
	if err != nil {
		return err
	}

	if err := DB.Delete(foodstock).Error; err != nil {
		log.Printf("Error deleting foodstock: %v", err)
		return err
	}

	log.Printf("✓ Foodstock %d deleted\n", foodID)
	return nil
}

// GetFoodstockByType ดึงแถวสต็อกของอาหารประเภทเดียว (เม็ดเล็ก/เม็ดใหญ่) ของผู้ใช้
// ตอนนี้ยังไม่มีที่ไหนเรียกใช้ (ใช้ GetFoodstockQuantity แทน)
func GetFoodstockByType(foodType string, userID int) (*models.Foodstock, error) {
	var foodstock models.Foodstock

	if err := DB.Where("food_type = ? AND user_id = ?", foodType, userID).First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ไม่พบข้อมูลสต็อกอาหารประเภท %s", foodType)
		}
		return nil, err
	}

	return &foodstock, nil
}

// EnsureFoodstockRows = migration ใช้ครั้งเดียว สำหรับข้อมูลรุ่นเก่า
// สมัยก่อนตาราง foodstock ยังไม่มี food_type และ user_id ฟังก์ชันนี้จะหาแถวเก่าที่ไม่มีประเภท
// แล้วกำหนดให้เป็น "เม็ดเล็ก" (จากนั้น migration ส่วนอื่นจะกำหนดเจ้าของให้)
// ถ้าไม่มีแถวเก่าเหลือแล้ว ก็ไม่ทำอะไร ปัจจุบันผู้ใช้ใหม่จะได้แถวของตัวเองตอนนำเข้าอาหารครั้งแรก
func EnsureFoodstockRows() error {
	var legacy models.Foodstock
	hasLegacyRow := DB.Where("food_type = ? OR food_type IS NULL", "").First(&legacy).Error == nil
	if !hasLegacyRow {
		return nil
	}

	legacy.FoodType = models.FoodTypeSmallPellet
	if err := DB.Save(&legacy).Error; err != nil {
		return err
	}
	log.Printf("✓ Migrated legacy foodstock row onto %s (%.2f kg)\n", models.FoodTypeSmallPellet, legacy.QuantityCurrent)
	return nil
}

// GetFoodstockQuantity คืนยอดคงเหลือ (กก.) ของอาหารประเภทนี้ของผู้ใช้ และบอกว่ามีแถวหรือไม่ (found)
// found = false แปลว่าฟาร์มนี้ไม่เคยมีอาหารประเภทนี้เลย
// ถูกเรียกจาก: handlers/food_distribution.go (เช็กว่าสต็อกหมดหรือยัง เช่นข้อความ "สต็อกหมดแล้ว ไม่มีอะไรให้ตัด")
func GetFoodstockQuantity(userID int, foodType string) (quantity float64, found bool, err error) {
	var foodstock models.Foodstock
	if err := DB.Where("food_type = ? AND user_id = ?", foodType, userID).First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, false, nil
		}
		return 0, false, err
	}
	return foodstock.QuantityCurrent, true, nil
}

// DeductFoodstockByType ตัดสต็อกอาหารประเภทนี้ลง amount กก. แบบ "ทั้งหมดหรือไม่ตัดเลย"
//
// คืนค่า: deducted (ตัดไปจริงกี่ กก.), shortfall (ขาดอีกกี่ กก.), err
//   - สต็อกพอ  -> ลดยอดลง amount, อัปเดต DateUp, คืน deducted = amount
//   - สต็อกไม่พอ -> "ไม่ตัดเลย" คืน deducted = 0 และ shortfall = ส่วนที่ขาด
//
// ทำไมไม่ตัดเท่าที่มี: เดิมเคยตัดบางส่วนจนยอดเหลือ 0 ทั้งที่อาหารจริงยังไม่ได้ถูกใช้ ทำให้ยอดในระบบ
// ไม่ตรงกับของจริง ผู้ใช้จึงต้องการว่าถ้าไม่พอก็ไม่ตัด รอเติมของก่อน
// ถูกเรียกจาก: handlers/foodstock.go (ตัดสต็อกอัตโนมัติ 18:00 น. และปุ่ม "ตัดสต็อก") และ handlers/food_distribution.go
func DeductFoodstockByType(foodType string, userID int, amount float64) (deducted float64, shortfall float64, err error) {
	var foodstock models.Foodstock

	if err := DB.Where("food_type = ? AND user_id = ?", foodType, userID).First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, 0, fmt.Errorf("ไม่พบข้อมูลสต็อกอาหารประเภท %s", foodType)
		}
		return 0, 0, fmt.Errorf("ดึงข้อมูลสต็อกล้มเหลว: %v", err)
	}

	if amount > foodstock.QuantityCurrent {
		shortfall = amount - foodstock.QuantityCurrent
		log.Printf("⚠️ สต็อกอาหาร %s ไม่พอ ต้องการ %.2f kg มีอยู่ %.2f kg (ขาด %.2f kg) ไม่ตัดสต็อก (user %d)\n", foodType, amount, foodstock.QuantityCurrent, shortfall, userID)
		return 0, shortfall, nil
	}

	foodstock.QuantityCurrent -= amount
	foodstock.DateUp = time.Now()

	if err := DB.Save(&foodstock).Error; err != nil {
		return 0, 0, fmt.Errorf("อัปเดตสต็อกล้มเหลว: %v", err)
	}

	log.Printf("✓ ตัดสต็อก %s %.2f kg คงเหลือ %.2f kg (user %d)\n", foodType, amount, foodstock.QuantityCurrent, userID)
	return amount, 0, nil
}
