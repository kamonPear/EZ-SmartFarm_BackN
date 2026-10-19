// =====================================================================================
// ไฟล์ database/importfood_repository.go - การนำอาหารเข้าคลัง (ล็อตอาหาร)
// =====================================================================================
//
// คลังอาหารใช้ 2 ตารางคู่กัน:
//   - import_food : ประวัติแต่ละครั้งที่นำอาหารเข้า (ล็อต) -> แสดงใน "ประวัติการนำเข้าอาหาร"
//   - foodstock   : ยอดคงเหลือรวมปัจจุบันของแต่ละประเภท (เม็ดเล็ก/เม็ดใหญ่) -> "สรุปสต็อกปัจจุบัน"
//
// ทุกครั้งที่นำเข้าล็อตใหม่ จะบวกยอดเข้า foodstock ให้อัตโนมัติ
// ถูกเรียกจาก: handlers/importfood.go และ handlers/foodstock.go
package database

import (
	"errors"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateImportFood บันทึกการนำอาหารเข้าคลัง (ปุ่ม "เข้าสต็อกอาหาร" ทั้งเว็บและแอป)
//
// ขั้นตอน:
//  1. สร้างแถวล็อตใหม่ในตาราง import_food (ประเภท, ปริมาณ กก., วันที่นำเข้า = ตอนนี้, วันหมดอายุ)
//  2. addToFoodstock บวกปริมาณเข้ายอดคงเหลือของประเภทนั้น
//
// ถูกเรียกจาก: handlers/importfood.go -> CreateImportFoodHandler (POST /api/importfoods)
func CreateImportFood(req *models.CreateImportFoodRequest, userID int) (*models.ImportFood, error) {
	lot := &models.ImportFood{
		UserID:       userID,
		FoodType:     req.FoodType,
		ImportVolume: req.ImportVolume,
		ImportDate:   time.Now(),
		ExpiryDate:   req.ExpiryDate,
	}

	if err := DB.Create(lot).Error; err != nil {
		log.Printf("Error creating import food lot: %v", err)
		return nil, err
	}

	if err := addToFoodstock(req.FoodType, userID, float64(req.ImportVolume)); err != nil {
		log.Printf("Error updating foodstock total after import: %v", err)
		return nil, err
	}

	// The threshold is set once and then locked: if it was already set, the value sent
	// with this import is ignored. The stock itself was already added, so failing to save
	// the threshold shouldn't undo that.
	if req.MinQuantity != nil && *req.MinQuantity > 0 {
		if err := SetFoodMinQuantity(userID, *req.MinQuantity); err != nil && !errors.Is(err, ErrMinQuantityLocked) {
			log.Printf("Warning: could not save min quantity after import: %v", err)
		}
	}

	log.Printf("✓ Import food lot #%d created (%s +%d kg) for user %d\n", lot.LotID, req.FoodType, req.ImportVolume, userID)
	return lot, nil
}

// addToFoodstock บวก amount (กก.) เข้ายอดคงเหลือของอาหารประเภท foodType ของผู้ใช้
//   - มีแถวของประเภทนี้อยู่แล้ว -> บวกเพิ่มและอัปเดตวันที่ DateUp
//   - ยังไม่มี (นำเข้าครั้งแรก) -> สร้างแถวใหม่ที่มียอดเท่ากับ amount
//
// ใช้ภายในไฟล์นี้เท่านั้น (ตัวพิมพ์เล็ก = private)
func addToFoodstock(foodType string, userID int, amount float64) error {
	var foodstock models.Foodstock

	err := DB.Where("food_type = ? AND user_id = ?", foodType, userID).First(&foodstock).Error
	if err == nil {
		foodstock.QuantityCurrent += amount
		foodstock.DateUp = time.Now()
		return DB.Save(&foodstock).Error
	}

	if err != gorm.ErrRecordNotFound {
		return err
	}

	foodstock = models.Foodstock{
		UserID:          userID,
		FoodType:        foodType,
		QuantityCurrent: amount,
		DateUp:          time.Now(),
		MinQuantity:     getSharedMinQuantity(userID),
	}
	return DB.Create(&foodstock).Error
}

// GetImportFoodByID ดึงล็อตนำเข้าล็อตเดียวตาม lotID เฉพาะของผู้ใช้คนนั้น ไม่เจอคืน ErrNotFound
// ถูกเรียกจาก: handlers/importfood.go (GET /api/importfoods?id=)
func GetImportFoodByID(lotID int, userID int) (*models.ImportFood, error) {
	var lot models.ImportFood
	if err := DB.Where("lot_id = ? AND user_id = ?", lotID, userID).First(&lot).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &lot, nil
}

// GetAllImportFood ดึงประวัติการนำเข้าทั้งหมดของผู้ใช้ เรียงจากล็อตล่าสุด
// ถูกเรียกจาก: handlers/importfood.go (แสดงประวัติ) และ handlers/foodstock.go (คำนวณสต็อก)
func GetAllImportFood(userID int) ([]models.ImportFood, error) {
	var lots []models.ImportFood
	if err := DB.Where("user_id = ?", userID).Order("lot_id desc").Find(&lots).Error; err != nil {
		log.Printf("Error fetching import food history: %v", err)
		return nil, err
	}
	return lots, nil
}

// DeleteAllImportFood ล้างคลังอาหารของผู้ใช้: ลบประวัติล็อตทั้งหมด "และ" ยอดคงเหลือใน foodstock
// ต้องลบ foodstock ด้วย เพราะยอดคงเหลือเป็นตัวเลขสะสมแยกต่างหาก ถ้าไม่ลบ ยอดเก่าจะค้างอยู่
// กรอง user_id เสมอ เพื่อไม่ให้ลบข้อมูลของฟาร์มอื่น
// ถูกเรียกจาก: handlers/importfood.go (DELETE /api/importfoods)
func DeleteAllImportFood(userID int) error {
	if err := DB.Where("user_id = ?", userID).Delete(&models.ImportFood{}).Error; err != nil {
		log.Printf("Error deleting import food history: %v", err)
		return err
	}
	// ลบ lot ประวัติไปแล้วแต่ยอดรวมคงเหลือ (foodstock) เป็นตัวเลขแยกต่างหากที่สะสมไว้
	// เองจาก addToFoodstock - ถ้าไม่ล้างตรงนี้ด้วย ยอดเก่าจะค้างอยู่ถาวรแม้ลบ lot
	// ไปหมดแล้ว (เคยพบว่า user ใหม่/ที่เพิ่งลบ lot เห็นยอดคงเหลือเก่าค้างอยู่)
	if err := DB.Where("user_id = ?", userID).Delete(&models.Foodstock{}).Error; err != nil {
		log.Printf("Error clearing foodstock totals: %v", err)
		return err
	}
	log.Println("✓ Import food history and foodstock totals cleared")
	return nil
}
