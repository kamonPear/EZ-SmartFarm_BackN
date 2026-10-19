// =====================================================================================
// ไฟล์ database/egg_repository.go - บันทึกการเก็บไข่ (ตาราง egg)
// =====================================================================================
//
// บันทึก 1 ครั้ง = 1 แถว: คอก, วันที่เก็บไข่, จำนวนไข่ (ฟอง), หมายเหตุ
// กฎสำคัญ 2 ข้อ:
//  1. คอกเดียวกันบันทึกซ้ำวันเดียวกันไม่ได้ (unique key: coop_id + date_collect_egg)
//  2. ไก่ต้องอายุครบ MinEggAgeMonths (5 เดือน) นับจากวันเกิดไก่ จึงบันทึกไข่ได้
//     เพราะไก่ไข่เริ่มออกไข่ราว 5 เดือน กันการกรอกวันที่ผิด
//
// ถูกเรียกจาก: handlers/egg.go
package database

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// IsDuplicateEggDate ตรวจว่า error จาก DB เกิดจาก "บันทึกไข่ซ้ำวันเดิมของคอกเดิม" หรือไม่
// (ข้อความ error มีทั้งคำว่า duplicate และ coop_id ซึ่งมาจาก unique key ของตาราง egg)
// handler จะได้ตอบผู้ใช้เป็นข้อความที่เข้าใจได้ว่า "วันนี้บันทึกไข่ของคอกนี้ไปแล้ว"
// ถูกเรียกจาก: handlers/egg.go
func IsDuplicateEggDate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") && strings.Contains(msg, "coop_id")
}

// MinEggAgeMonths อายุไก่ขั้นต่ำ (เดือน นับจากวันเกิดไก่ของคอก) ก่อนจะบันทึกการเก็บไข่ได้
// ใช้ 5 เดือน เพราะเป็นช่วงที่ไก่ไข่เริ่มให้ไข่ ตั้งเป็นค่าคงที่ไว้ที่เดียวเพื่อแก้ง่าย
const MinEggAgeMonths = 5

// ChickensTooYoungError error แบบพิเศษ: บันทึกไข่ของวันที่ไก่ยังอายุไม่ถึง MinEggAgeMonths
// EligibleDate คือวันแรก (YYYY-MM-DD) ที่คอกนี้บันทึกไข่ได้
// ทำไมต้องสร้าง type เอง: handler จะได้ดึงวันที่นี้ไปบอกผู้ใช้ได้ว่า "เก็บไข่ได้ตั้งแต่วันที่ ..."
type ChickensTooYoungError struct {
	EligibleDate string
}

// Error ทำให้ ChickensTooYoungError เป็น error ของ Go ได้ คืนข้อความภาษาไทยพร้อมวันที่เริ่มเก็บไข่ได้
func (e *ChickensTooYoungError) Error() string {
	return fmt.Sprintf("ไก่ยังอายุไม่ถึง %d เดือน เก็บไข่ได้ตั้งแต่วันที่ %s", MinEggAgeMonths, e.EligibleDate)
}

// IsChickensTooYoung ตรวจว่า err เป็น ChickensTooYoungError หรือไม่ ถ้าใช่ คืนตัว error มาด้วย
// ใช้ errors.As ซึ่งแกะ error ที่ถูกห่อไว้ได้ (ปลอดภัยกว่าการเทียบ type ตรง ๆ)
// ถูกเรียกจาก: handlers/egg.go (เพื่อตอบ 422 พร้อมวันที่ EligibleDate)
func IsChickensTooYoung(err error) (*ChickensTooYoungError, bool) {
	var tooYoung *ChickensTooYoungError
	if errors.As(err, &tooYoung) {
		return tooYoung, true
	}
	return nil, false
}

// checkChickensOldEnough ตรวจว่าวันที่เก็บไข่ ไม่อยู่ก่อน (วันเกิดไก่ + MinEggAgeMonths เดือน)
//  1. ดึงเฉพาะ coop_id และ birthday ของคอก (Select เท่าที่ใช้ ประหยัดกว่า)
//  2. ถ้าคอกไม่มีวันเกิดไก่ -> ผ่าน เพราะคำนวณอายุไม่ได้
//  3. เทียบวันแบบข้อความ YYYY-MM-DD (models.DateKey) เพื่อไม่ให้เวลา/เขตเวลามีผล
//
// ถูกเรียกจาก: CreateEgg และ UpdateEgg ในไฟล์นี้
func checkChickensOldEnough(coopID int, collectDate time.Time) error {
	var coop models.Coop
	if err := DB.Select("coop_id", "birthday").First(&coop, coopID).Error; err != nil {
		return err
	}
	if coop.Birthday.IsZero() {
		return nil
	}
	eligible := models.DateKey(coop.Birthday.AddDate(0, MinEggAgeMonths, 0))
	if models.DateKey(collectDate) < eligible {
		return &ChickensTooYoungError{EligibleDate: eligible}
	}
	return nil
}

// CreateEgg บันทึกการเก็บไข่ใหม่ (หน้า "บันทึกการเก็บไข่")
//  1. เช็กว่าคอกเป็นของผู้ใช้ (CoopBelongsToUser) ไม่ใช่ -> ErrNotFound
//  2. เช็กอายุไก่ (checkChickensOldEnough)
//  3. บันทึกลง DB (ถ้าซ้ำวันเดิม DB คืน error ซ้ำ ซึ่ง handler แยกด้วย IsDuplicateEggDate)
//  4. โหลดแถวใหม่พร้อมข้อมูลคอก (Preload) เพื่อส่งกลับให้แอปแสดงชื่อคอกได้ทันที
//
// ถูกเรียกจาก: handlers/egg.go -> CreateEggHandler (POST /api/eggs)
func CreateEgg(req *models.CreateEggRequest, userID int) (*models.Egg, error) {
	owned, err := CoopBelongsToUser(req.CoopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	if err := checkChickensOldEnough(req.CoopID, req.DateCollectEgg); err != nil {
		return nil, err
	}

	egg := &models.Egg{
		CoopID:         req.CoopID,
		DateCollectEgg: req.DateCollectEgg,
		NumberEgg:      req.NumberEgg,
		Note:           req.Note,
	}

	if err := DB.Create(egg).Error; err != nil {
		log.Printf("Error creating egg record: %v", err)
		return nil, err
	}

	if err := DB.Preload("Coop").First(egg, egg.EggID).Error; err != nil {
		log.Printf("Error reloading created egg record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Egg record created with ID: %d for Coop: %d\n", egg.EggID, egg.CoopID)
	return egg, nil
}

// GetEggByID ดึงบันทึกไข่ 1 รายการตาม id พร้อมข้อมูลคอก
// ใช้ subquery "coop_id IN (คอกของผู้ใช้)" เพื่อให้เห็นเฉพาะของตัวเอง ไม่เจอคืน ErrNotFound
// ถูกใช้ใน UpdateEgg / DeleteEgg และ handlers/egg.go
func GetEggByID(eggID int, userID int) (*models.Egg, error) {
	var egg *models.Egg

	if err := DB.Preload("Coop").
		Where("egg_id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", eggID, userID).
		First(&egg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error fetching egg record: %v", err)
		return nil, err
	}

	return egg, nil
}

// GetEggsByCoopID ดึงบันทึกไข่ทั้งหมดของคอกเดียว เรียงจากวันล่าสุด (หลังเช็กว่าคอกเป็นของผู้ใช้)
// ใช้แสดงประวัติการเก็บไข่รายคอก
func GetEggsByCoopID(coopID int, userID int) ([]models.Egg, error) {
	var eggs []models.Egg

	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	if err := DB.Preload("Coop").
		Where("coop_id = ?", coopID).
		Order("date_collect_egg DESC").
		Find(&eggs).Error; err != nil {
		log.Printf("Error fetching egg records: %v", err)
		return nil, err
	}

	return eggs, nil
}

// GetAllEggs ดึงบันทึกไข่ของทุกคอกของผู้ใช้ เรียงจากวันล่าสุด
// ใช้ทำกราฟ/สรุปจำนวนไข่ในหน้าหลักและหน้ารายงาน
func GetAllEggs(userID int) ([]models.Egg, error) {
	var eggs []models.Egg

	if err := DB.Preload("Coop").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Order("date_collect_egg DESC").
		Find(&eggs).Error; err != nil {
		log.Printf("Error fetching all egg records: %v", err)
		return nil, err
	}

	return eggs, nil
}

// UpdateEgg แก้ไขบันทึกไข่ (หน้าต่าง "แก้ไข บันทึกไข่")
//  1. เช็กว่าบันทึกเป็นของผู้ใช้ (GetEggByID)
//  2. ถ้าย้ายไปคอกอื่น คอกปลายทางต้องเป็นของผู้ใช้ด้วย
//  3. ถ้าเปลี่ยนคอกหรือวันที่ ต้องเช็กอายุไก่ใหม่ ด้วยคู่ (คอก, วัน) หลังแก้
//  4. อัปเดตเฉพาะช่องที่ส่งมา (map ชื่อคอลัมน์จริง) ถ้าไม่มีอะไรเปลี่ยนคืนค่าเดิม
func UpdateEgg(eggID int, req *models.UpdateEggRequest, userID int) (*models.Egg, error) {
	egg, err := GetEggByID(eggID, userID)
	if err != nil {
		return nil, err
	}

	// If CoopID is provided, verify the target coop exists and belongs to userID too
	if req.CoopID > 0 && req.CoopID != egg.CoopID {
		owned, err := CoopBelongsToUser(req.CoopID, userID)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, ErrNotFound
		}
	}

	// ย้ายคอก/เปลี่ยนวัน ก็ต้องผ่านเกณฑ์อายุไก่ของ (คอก, วัน) ใหม่เช่นกัน
	if req.CoopID > 0 || !req.DateCollectEgg.IsZero() {
		targetCoop := egg.CoopID
		if req.CoopID > 0 {
			targetCoop = req.CoopID
		}
		targetDate := egg.DateCollectEgg
		if !req.DateCollectEgg.IsZero() {
			targetDate = req.DateCollectEgg
		}
		if err := checkChickensOldEnough(targetCoop, targetDate); err != nil {
			return nil, err
		}
	}

	// Update only provided fields
	updates := map[string]interface{}{}

	if req.CoopID > 0 {
		updates["coop_id"] = req.CoopID
	}
	if !req.DateCollectEgg.IsZero() {
		updates["date_collect_egg"] = req.DateCollectEgg
	}
	if req.NumberEgg > 0 {
		updates["number_egg"] = req.NumberEgg
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if len(updates) == 0 {
		return egg, nil
	}

	if err := DB.Model(egg).Updates(updates).Error; err != nil {
		log.Printf("Error updating egg record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Egg record %d updated\n", eggID)
	return egg, nil
}

// DeleteEgg ลบบันทึกไข่ 1 รายการ (เช็กสิทธิ์ผ่าน GetEggByID ก่อน)
func DeleteEgg(eggID int, userID int) error {
	egg, err := GetEggByID(eggID, userID)
	if err != nil {
		return err
	}

	if err := DB.Delete(egg).Error; err != nil {
		log.Printf("Error deleting egg record: %v", err)
		return err
	}

	fmt.Printf("✓ Egg record %d deleted\n", eggID)
	return nil
}

// DeleteEggsByCoopID ลบบันทึกไข่ทั้งหมดของคอก "โดยไม่เช็กเจ้าของ" ตอนนี้ไม่มีที่ไหนเรียกใช้
// (การลบคอกใช้ Transaction ใน DeleteCoop แทน)
func DeleteEggsByCoopID(coopID int) error {
	if err := DB.Where("coop_id = ?", coopID).Delete(&models.Egg{}).Error; err != nil {
		log.Printf("Error deleting egg records for coop %d: %v", coopID, err)
		return err
	}

	fmt.Printf("✓ All egg records for coop %d deleted\n", coopID)
	return nil
}
