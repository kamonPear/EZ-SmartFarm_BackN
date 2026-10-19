// =====================================================================================
// ไฟล์ database/food_age_rule_repository.go - ตารางปริมาณอาหารตามอายุไก่ (ตาราง food_age_rule)
// =====================================================================================
//
// ผู้ใช้แต่ละคนตั้ง "ตารางให้อาหาร" ของฟาร์มตัวเองได้ แต่ละแถวบอกว่า
// ไก่อายุตั้งแต่ min_weeks ถึง max_weeks สัปดาห์ กินอาหารประเภทไหน (เม็ดเล็ก/เม็ดใหญ่) ตัวละกี่กรัมต่อวัน
// ตารางนี้ถูกใช้คำนวณ "อาหารที่ใช้ต่อวัน" ของแต่ละคอก -> ใช้ตัดสต็อกอัตโนมัติและแสดงในหน้าคลังอาหาร
// ถูกเรียกจาก: handlers/food_age_rule.go และ handlers/foodstock.go
package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"
)

// GetFoodAgeRules ดึงตารางให้อาหารทั้งหมดของผู้ใช้ เรียงตามอายุเริ่มต้น (min_weeks) จากน้อยไปมาก
// เรียงแบบนี้เพื่อให้หน้าเว็บแสดงตามลำดับช่วงอายุ และหาช่วงอายุที่ตรงกับไก่ได้ง่าย
// ถ้าผู้ใช้ยังไม่เคยตั้งค่า จะได้ list ว่าง (ไม่ใช่ error)
// ถูกเรียกจาก: handlers/food_age_rule.go (แสดงตาราง) และ handlers/foodstock.go (คำนวณอาหารต่อวัน/ตัดสต็อก)
func GetFoodAgeRules(userID int) (models.FoodRules, error) {
	rules := models.FoodRules{}
	if err := DB.Where("user_id = ?", userID).Order("min_weeks ASC").Find(&rules).Error; err != nil {
		log.Printf("Error fetching food age rules: %v", err)
		return nil, err
	}
	return rules, nil
}

// ReplaceFoodAgeRules บันทึกตารางให้อาหารใหม่ "แทนที่ทั้งตาราง" ใน Transaction เดียว
//  1. ลบแถวเดิมทั้งหมดของผู้ใช้
//  2. เพิ่มแถวใหม่ตามที่ส่งมาทีละแถว
//  3. Commit แล้วดึงตารางใหม่กลับไปให้หน้าเว็บ
//
// ทำไมลบแล้วเพิ่มใหม่: หน้าต่างแก้ไขบนเว็บแก้ตารางเป็นรายการทั้งก้อน (เพิ่ม/ลบ/แก้หลายแถวพร้อมกัน)
// แทนที่ทั้งชุดง่ายกว่าและพลาดน้อยกว่าการไล่เทียบทีละแถว
// ทำไมใช้ Transaction: ถ้าเพิ่มแถวไหนล้ม จะ Rollback กลับเป็นตารางเดิม ไม่เหลือตารางครึ่ง ๆ กลาง ๆ
// ถูกเรียกจาก: handlers/food_age_rule.go (PUT)
func ReplaceFoodAgeRules(userID int, rows []models.FoodAgeRuleInput) (models.FoodRules, error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.FoodAgeRule{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error clearing food age rules: %v", err)
		return nil, err
	}
	for _, in := range rows {
		rule := models.FoodAgeRule{
			UserID:       userID,
			FoodType:     in.FoodType,
			MinWeeks:     in.MinWeeks,
			MaxWeeks:     in.MaxWeeks,
			GramsPerBird: in.GramsPerBird,
		}
		if err := tx.Create(&rule).Error; err != nil {
			tx.Rollback()
			log.Printf("Error saving food age rule: %v", err)
			return nil, err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return GetFoodAgeRules(userID)
}
