package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFoodstockByID retrieves a foodstock row by ID, only if it's owned by userID
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

// GetAllFoodstocks retrieves every foodstock row owned by userID
func GetAllFoodstocks(userID int) ([]models.Foodstock, error) {
	var foodstocks []models.Foodstock

	if err := DB.Where("user_id = ?", userID).Find(&foodstocks).Error; err != nil {
		log.Printf("Error fetching foodstocks: %v", err)
		return nil, err
	}

	return foodstocks, nil
}

// UpdateFoodstock manually corrects the current stock total, only if it's owned by userID
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

// DeleteFoodstock deletes a foodstock row, only if it's owned by userID
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

// GetFoodstockByType retrieves userID's running total row for a single food type (เม็ดเล็ก/เม็ดใหญ่)
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

// EnsureFoodstockRows is a one-time migration that adopts the legacy pre-ownership
// untyped row (food_id=1, no food_type, no owner) onto the small pellet type, back
// when foodstock had no user_id FK yet (it gets backfilled onto the bootstrap admin
// user by ensureAdminBootstrapAndOwnership right after this runs). That FK now exists
// on every environment, so it's no longer safe to create an ownerless row here even
// as a fallback - every user gets their own per-type rows lazily instead (see
// addToFoodstock the first time they import food). If there's no legacy row left to
// adopt (already migrated, or the table was cleared out), this is a no-op.
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

// GetFoodstockQuantity returns userID's current quantity_current for foodType, and
// whether a row exists at all yet (found=false if the farm never had this food type).
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

// DeductFoodstockByType ลด quantity_current ของอาหารประเภทที่ระบุลงตามจำนวนที่ขอ
// (amount) เฉพาะสต็อกของ userID เท่านั้น - ตัดแบบ "ทั้งหมดหรือไม่เลย" เท่านั้น ถ้า
// สต็อกที่เหลือไม่พอสำหรับ amount ที่ขอ จะไม่ตัดอะไรเลยแม้แต่กิโลเดียว (deducted=0)
// แล้วรายงานส่วนที่ขาด (shortfall) กลับไปแทน - เดิมฟังก์ชันนี้ตัดสต็อกเท่าที่มีให้
// บางส่วนแล้วค่อยรายงานว่าขาด ทำให้สต็อกที่เหลือน้อยอยู่แล้วถูกล้างเป็น 0 ไปเงียบๆ
// ทั้งที่ของจริงยังไม่ได้ถูกเอาออกไปใช้แต่อย่างใด (ผู้ใช้ยืนยันว่าไม่ต้องการแบบนั้น -
// ถ้าไม่พอสำหรับยอดที่ต้องการจริง ก็ไม่ควรตัดอะไรเลย ปล่อยให้สต็อกที่เหลือยังอยู่ครบ
// จนกว่าจะเติมของเพิ่ม)
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

