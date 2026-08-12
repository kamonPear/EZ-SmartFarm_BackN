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

// DeductDailyFoodstock ทำหน้าที่ลด quantity_current ลงตามจำนวนที่กำหนด (เช่น วันละ 20)
func DeductDailyFoodstock(amount float64) error {
	var foodstock models.Foodstock

	// 1. ดึงข้อมูลสต็อก ID 1
	if err := DB.First(&foodstock, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ไม่พบข้อมูลสต็อกอาหาร (ID=1)")
		}
		return fmt.Errorf("ดึงข้อมูลสต็อกล้มเหลว: %v", err)
	}

	// 2. เช็คว่ามีอาหารเหลือให้ตัดหรือไม่
	if foodstock.QuantityCurrent <= 0 {
		log.Println("⚠️ สต็อกอาหารปัจจุบันเป็น 0 ไม่สามารถตัดสต็อกเพิ่มได้")
		return nil
	}

	// 3. หักลบจำนวนอาหาร
	foodstock.QuantityCurrent -= amount

	// 4. ดักจับกรณีหักแล้วยอดติดลบ ให้เซ็ตเป็น 0
	if foodstock.QuantityCurrent < 0 {
		foodstock.QuantityCurrent = 0
	}
	foodstock.DateUp = time.Now()

	// 5. บันทึกข้อมูลกลับลง Database
	if err := DB.Save(&foodstock).Error; err != nil {
		return fmt.Errorf("อัปเดตสต็อกล้มเหลว: %v", err)
	}

	log.Printf("✓ ตัดสต็อกอัตโนมัติ %.2f kg คงเหลือ %.2f kg\n", amount, foodstock.QuantityCurrent)
	return nil
}
