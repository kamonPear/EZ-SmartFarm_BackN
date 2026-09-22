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

// EnsureFoodstockRows makes sure every known food type has a foodstock row for the
// legacy/original data (no owner yet - gets backfilled onto the bootstrap admin user
// by ensureAdminBootstrapAndOwnership during migration). Existing databases only ever
// had a single untyped row (food_id=1) - its quantity is carried over onto the small
// pellet type here, and the large pellet type starts at 0. New users get their own
// per-type rows lazily (see addToFoodstock) the first time they import food.
func EnsureFoodstockRows() error {
	var legacy models.Foodstock
	hasLegacyRow := DB.Where("food_type = ? OR food_type IS NULL", "").First(&legacy).Error == nil

	for _, foodType := range []string{models.FoodTypeSmallPellet, models.FoodTypeLargePellet} {
		// Deliberately checks for ANY row of this food type (any owner), not just
		// unowned ones - once the legacy row has been split into typed rows the first
		// time, this must never fire again, otherwise every subsequent boot would try
		// to insert another ownerless (user_id=0) row here, which the user_id FK added
		// by ensureAdminBootstrapAndOwnership rejects. New users get their own rows
		// lazily via addToFoodstock instead.
		var count int64
		if err := DB.Model(&models.Foodstock{}).Where("food_type = ?", foodType).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		if hasLegacyRow && foodType == models.FoodTypeSmallPellet {
			legacy.FoodType = foodType
			if err := DB.Save(&legacy).Error; err != nil {
				return err
			}
			log.Printf("✓ Migrated legacy foodstock row onto %s (%.2f kg)\n", foodType, legacy.QuantityCurrent)
			continue
		}

		if err := DB.Create(&models.Foodstock{
			FoodType:        foodType,
			QuantityCurrent: 0,
			DateUp:          time.Now(),
		}).Error; err != nil {
			return err
		}
		log.Printf("✓ Created foodstock row for %s\n", foodType)
	}

	return nil
}

// DeductFoodstockByType ทำหน้าที่ลด quantity_current ของอาหารประเภทที่ระบุลงตามจำนวนที่กำหนด
// เฉพาะสต็อกของ userID เท่านั้น
func DeductFoodstockByType(foodType string, userID int, amount float64) error {
	var foodstock models.Foodstock

	if err := DB.Where("food_type = ? AND user_id = ?", foodType, userID).First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ไม่พบข้อมูลสต็อกอาหารประเภท %s", foodType)
		}
		return fmt.Errorf("ดึงข้อมูลสต็อกล้มเหลว: %v", err)
	}

	// เช็คว่ามีอาหารเหลือให้ตัดหรือไม่
	if foodstock.QuantityCurrent <= 0 {
		log.Printf("⚠️ สต็อกอาหาร %s ปัจจุบันเป็น 0 ไม่สามารถตัดสต็อกเพิ่มได้\n", foodType)
		return nil
	}

	// หักลบจำนวนอาหาร
	foodstock.QuantityCurrent -= amount

	// ดักจับกรณีหักแล้วยอดติดลบ ให้เซ็ตเป็น 0
	if foodstock.QuantityCurrent < 0 {
		foodstock.QuantityCurrent = 0
	}
	foodstock.DateUp = time.Now()

	// บันทึกข้อมูลกลับลง Database
	if err := DB.Save(&foodstock).Error; err != nil {
		return fmt.Errorf("อัปเดตสต็อกล้มเหลว: %v", err)
	}

	log.Printf("✓ ตัดสต็อก %s %.2f kg คงเหลือ %.2f kg (user %d)\n", foodType, amount, foodstock.QuantityCurrent, userID)
	return nil
}

// DeductFoodstockByTypeAllUsers applies the fixed daily automatic deduction to every
// user's stock row for foodType. Used only by the cron scheduler (scheduler.SetupJobs),
// which runs on a timer with no authenticated caller/user in context - unlike
// DeductFoodstockByType (used by the per-user HTTP handlers), it isn't scoped to a
// single user because it represents farm-wide daily consumption applying equally to
// every account's own stock.
func DeductFoodstockByTypeAllUsers(foodType string, amount float64) error {
	var rows []models.Foodstock
	if err := DB.Where("food_type = ?", foodType).Find(&rows).Error; err != nil {
		return fmt.Errorf("ดึงข้อมูลสต็อกล้มเหลว: %v", err)
	}

	for i := range rows {
		if rows[i].QuantityCurrent <= 0 {
			continue
		}
		rows[i].QuantityCurrent -= amount
		if rows[i].QuantityCurrent < 0 {
			rows[i].QuantityCurrent = 0
		}
		rows[i].DateUp = time.Now()
		if err := DB.Save(&rows[i]).Error; err != nil {
			return fmt.Errorf("อัปเดตสต็อกล้มเหลว: %v", err)
		}
	}

	log.Printf("✓ ตัดสต็อก %s %.2f kg ให้ทุกผู้ใช้ (%d ราย) เรียบร้อยแล้ว\n", foodType, amount, len(rows))
	return nil
}
