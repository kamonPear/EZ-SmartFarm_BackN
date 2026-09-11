package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFoodstockByID retrieves a foodstock by ID
func GetFoodstockByID(foodID int) (*models.Foodstock, error) {
	var foodstock *models.Foodstock

	if err := DB.Where("food_id = ?", foodID).
		First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("foodstock not found")
		}
		log.Printf("Error fetching foodstock: %v", err)
		return nil, err
	}

	return foodstock, nil
}

// GetAllFoodstocks retrieves all foodstocks from the database
func GetAllFoodstocks() ([]models.Foodstock, error) {
	var foodstocks []models.Foodstock

	if err := DB.Find(&foodstocks).Error; err != nil {
		log.Printf("Error fetching foodstocks: %v", err)
		return nil, err
	}

	return foodstocks, nil
}

// UpdateFoodstock manually corrects the current stock total (ID is always locked to 1)
func UpdateFoodstock(id int, req *models.UpdateFoodstockRequest) (*models.Foodstock, error) {
	var foodstock models.Foodstock
	if err := DB.First(&foodstock, id).Error; err != nil {
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

// DeleteFoodstock deletes a foodstock from the database
func DeleteFoodstock(foodID int) error {
	foodstock, err := GetFoodstockByID(foodID)
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

// GetFoodstockByType retrieves the running total row for a single food type (เม็ดเล็ก/เม็ดใหญ่)
func GetFoodstockByType(foodType string) (*models.Foodstock, error) {
	var foodstock models.Foodstock

	if err := DB.Where("food_type = ?", foodType).First(&foodstock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ไม่พบข้อมูลสต็อกอาหารประเภท %s", foodType)
		}
		return nil, err
	}

	return &foodstock, nil
}

// EnsureFoodstockRows makes sure every known food type has a foodstock row.
// Existing databases only ever had a single untyped row (food_id=1) - its quantity is
// carried over onto the small pellet type here, and the large pellet type starts at 0.
func EnsureFoodstockRows() error {
	var legacy models.Foodstock
	hasLegacyRow := DB.Where("food_type = ? OR food_type IS NULL", "").First(&legacy).Error == nil

	for _, foodType := range []string{models.FoodTypeSmallPellet, models.FoodTypeLargePellet} {
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
func DeductFoodstockByType(foodType string, amount float64) error {
	var foodstock models.Foodstock

	if err := DB.Where("food_type = ?", foodType).First(&foodstock).Error; err != nil {
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

	log.Printf("✓ ตัดสต็อก %s %.2f kg คงเหลือ %.2f kg\n", foodType, amount, foodstock.QuantityCurrent)
	return nil
}
