// =====================================================================================
// ไฟล์ database/coop_repository.go - ข้อมูลคอกไก่ (ตาราง coop) ซึ่งเป็น "ตารางแม่" ของระบบ
// =====================================================================================
//
// เกือบทุกข้อมูลผูกกับคอก: ไข่ (egg), สุขภาพ (health), ประวัติวัคซีน (vaccine_history),
// อุปกรณ์ (device), นัดตรวจ (health_appointments) และคอกผูกกับผู้ใช้ผ่าน user_id
// การเช็กว่าผู้ใช้มีสิทธิ์กับข้อมูลไหน จึงมักเช็กผ่าน "คอกนี้เป็นของผู้ใช้คนนี้ไหม" (CoopBelongsToUser)
//
// คอลัมน์หลัก: ชื่อคอก, วันรับเข้าเลี้ยง, จำนวนไก่, วันเกิดไก่, หมายเหตุ, ตำแหน่งบนผังฟาร์ม (pos_x/pos_y)
// ถูกเรียกจาก: handlers/coop.go และ handler อื่นอีกหลายตัวที่ต้องใช้ข้อมูลคอก
package database

import (
	"fmt"
	"log"
	"strings"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// IsDuplicateNameCoop ตรวจว่า error จากฐานข้อมูลเกิดจาก "ชื่อคอกซ้ำ" หรือไม่
// ตาราง coop มี unique key (uq_coop_user_name) = ผู้ใช้คนเดียวกันตั้งชื่อคอกซ้ำไม่ได้
// ถ้าเพิ่มคอกชื่อซ้ำ MySQL จะคืน error "Duplicate entry ..." ฟังก์ชันนี้อ่านข้อความ error เพื่อแยกกรณีนี้
// handler จะได้ตอบผู้ใช้ว่า "ชื่อคอกนี้มีอยู่แล้ว" แทนข้อความ error ทั่วไป
// ถูกเรียกจาก: handlers/coop.go
func IsDuplicateNameCoop(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return (strings.Contains(msg, "name_coop") || strings.Contains(msg, "uq_coop_user_name")) &&
		(strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique"))
}

// CreateCoop เพิ่มคอกใหม่ให้ผู้ใช้ (หน้า "เพิ่มคอกไก่")
// การตรวจข้อมูล (ต้องมีชื่อ วันรับเข้าเลี้ยง วันเกิดไก่ จำนวนไก่ไม่เป็น 0 และวันรับเข้าเลี้ยงต้องไม่ก่อนวันเกิด)
// ทำไว้ใน handlers/coop.go ก่อนเรียกฟังก์ชันนี้แล้ว
// ถูกเรียกจาก: handlers/coop.go -> CreateCoopHandler (POST /api/coops)
func CreateCoop(req *models.CreateCoopRequest, userID int) (*models.Coop, error) {
	coop := &models.Coop{
		UserID:           userID,
		NameCoop:         req.NameCoop,
		DateAdoptAnimals: req.DateAdoptAnimals,
		Amount:           req.Amount,
		Birthday:         req.Birthday,
		Note:             req.Note,
	}

	if err := DB.Create(coop).Error; err != nil {
		log.Printf("Error creating coop: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Coop created with ID: %d\n", coop.CoopID)
	return coop, nil
}

// GetCoopByID ดึงคอกตาม id "โดยไม่เช็กเจ้าของ" พร้อมข้อมูลลูกทั้งหมด (Preload อุปกรณ์/ไข่/สุขภาพ/วัคซีน)
// ตอนนี้ไม่มีที่ไหนเรียกใช้แล้ว ถ้าต้องดึงคอกให้ผู้ใช้ ให้ใช้ GetCoopByIDForUser ที่เช็กเจ้าของ
func GetCoopByID(coopID int) (*models.Coop, error) {
	var coop *models.Coop

	if err := DB.Preload("Devices").
		Preload("Eggs").
		Preload("Health").
		Preload("VaccineHistory").
		Where("coop_id = ?", coopID).
		First(&coop).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("coop not found")
		}
		log.Printf("Error fetching coop: %v", err)
		return nil, err
	}

	return coop, nil
}

// GetCoopByIDForUser ดึงคอกตาม id เฉพาะที่เป็นของผู้ใช้ พร้อมข้อมูลลูก (Preload)
//
// คืน ErrNotFound ทั้งกรณี "ไม่มีคอกนี้" และ "คอกนี้เป็นของคนอื่น"
// ทำไมไม่แยก: ถ้าตอบต่างกัน ผู้ไม่หวังดีจะรู้ว่า id ไหนมีอยู่จริงในระบบ
// Preload = ดึงข้อมูลตารางลูกมาใส่ในคอกด้วยคำสั่งเดียว แทนการ query แยกทีละตาราง
// ถูกเรียกจาก: handlers/coop.go (GetCoopHandler ดูคอกเดียว) และ handlers/vaccine.go
func GetCoopByIDForUser(coopID, userID int) (*models.Coop, error) {
	var coop *models.Coop

	if err := DB.Preload("Devices").
		Preload("Eggs").
		Preload("Health").
		Preload("VaccineHistory").
		Where("coop_id = ? AND user_id = ?", coopID, userID).
		First(&coop).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error fetching coop: %v", err)
		return nil, err
	}

	return coop, nil
}

// GetAllCoops ดึงคอกทุกคอกของผู้ใช้ พร้อมอุปกรณ์ ไข่ ผลตรวจสุขภาพ และประวัติวัคซีนของแต่ละคอก
// ใช้แสดงหน้า "คอกไก่ทั้งหมด" และหน้าหลัก รวมถึงใช้คำนวณปฏิทิน/วัคซีน/อาหาร
// ถูกเรียกจาก: handlers/coop.go, calendar.go, vaccine.go, foodstock.go, coop_food_consumption.go
// และ GetMotionAlertsForUser ใน sensor_log_repository.go
func GetAllCoops(userID int) ([]models.Coop, error) {
	var coops []models.Coop

	if err := DB.Preload("Devices").
		Preload("Eggs").
		Preload("Health").
		Preload("VaccineHistory").
		Where("user_id = ?", userID).
		Find(&coops).Error; err != nil {
		log.Printf("Error fetching coops: %v", err)
		return nil, err
	}

	return coops, nil
}

// GetAllCoopsGroupedByUser ดึงคอกทั้งหมดในระบบ (ทุกผู้ใช้) แล้วจัดกลุ่มตาม user_id
// เป็น map[userID] -> รายการคอก ไม่ Preload ตารางลูก เพราะต้องใช้แค่จำนวนไก่และอายุ
//
// ใช้ทำอะไร: ตัวตัดสต็อกอาหารอัตโนมัติ 18:00 น. ต้องคำนวณอาหารที่ใช้ต่อวันของ "ทุกฟาร์ม"
// query ครั้งเดียวแล้วจัดกลุ่มเอง เร็วกว่าการ query ทีละผู้ใช้มาก
// ถูกเรียกจาก: handlers/foodstock.go
func GetAllCoopsGroupedByUser() (map[int][]models.Coop, error) {
	var coops []models.Coop
	if err := DB.Find(&coops).Error; err != nil {
		log.Printf("Error fetching all coops: %v", err)
		return nil, err
	}

	grouped := make(map[int][]models.Coop)
	for _, c := range coops {
		grouped[c.UserID] = append(grouped[c.UserID], c)
	}
	return grouped, nil
}

// UpdateCoop แก้ไขข้อมูลคอก (หน้า "แก้ไขข้อมูลคอกไก่") เฉพาะคอกของผู้ใช้
// อัปเดตเฉพาะช่องที่ส่งค่ามา (ชื่อไม่ว่าง, วันที่ไม่เป็นศูนย์, จำนวนไก่ > 0) ช่องอื่นคงเดิม
// ใช้ map กำหนดชื่อคอลัมน์ตรง ๆ ถ้าไม่มีอะไรเปลี่ยนเลยก็คืนคอกเดิมโดยไม่ยิงคำสั่งไป DB
// ถูกเรียกจาก: handlers/coop.go -> UpdateCoopHandler (PUT /api/coops)
func UpdateCoop(coopID int, req *models.UpdateCoopRequest, userID int) (*models.Coop, error) {
	coop, err := GetCoopByIDForUser(coopID, userID)
	if err != nil {
		return nil, err
	}

	// Update only provided fields
	updates := map[string]interface{}{}

	if req.NameCoop != "" {
		updates["name_coop"] = req.NameCoop
	}
	if !req.DateAdoptAnimals.IsZero() {
		updates["date_adopt_animals"] = req.DateAdoptAnimals
	}
	if req.Amount > 0 {
		updates["amount"] = req.Amount
	}
	if !req.Birthday.IsZero() {
		updates["birthday"] = req.Birthday
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if len(updates) == 0 {
		return coop, nil
	}

	if err := DB.Model(coop).Updates(updates).Error; err != nil {
		log.Printf("Error updating coop: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Coop %d updated\n", coopID)
	return coop, nil
}

// UpdateCoopPositions บันทึกตำแหน่งของทุกคอกบนผังฟาร์มในครั้งเดียว (หน้า "จัดวางผังฟาร์ม")
//
// ขั้นตอน (ใน Transaction):
//  1. เช็กว่าทุกคอกในรายการเป็นของผู้ใช้ ถ้ามีคอกไหนไม่ใช่ ยกเลิกทั้งชุด (ErrNotFound) ก่อนเขียนอะไรลงไป
//  2. อัปเดต pos_x / pos_y ทีละคอก
//
// ทำไมใช้ map ตรง ๆ แทน UpdateCoop: ตำแหน่ง (0,0) หรือ nil (เอาคอกออกจากผัง) เป็นค่าที่ต้องบันทึกจริง
// ถ้าใช้วิธีข้ามค่าศูนย์แบบ UpdateCoop ค่าพวกนี้จะไม่ถูกบันทึก
// ถูกเรียกจาก: handlers/farm_layout.go (PUT /api/coops/positions)
func UpdateCoopPositions(entries []models.CoopPositionEntry, userID int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, e := range entries {
			ok, err := CoopBelongsToUser(e.CoopID, userID)
			if err != nil {
				return err
			}
			if !ok {
				return ErrNotFound
			}
		}
		for _, e := range entries {
			if err := tx.Model(&models.Coop{}).
				Where("coop_id = ? AND user_id = ?", e.CoopID, userID).
				Updates(map[string]interface{}{"pos_x": e.PosX, "pos_y": e.PosY}).Error; err != nil {
				log.Printf("Error updating position for coop %d: %v", e.CoopID, err)
				return err
			}
		}
		return nil
	})
}

// DeleteCoop ลบคอกและข้อมูลที่เกี่ยวข้องทั้งหมด (เฉพาะคอกของผู้ใช้)
// ใช้ Transaction ลบตามลำดับ: อุปกรณ์ -> ไข่ -> ผลตรวจสุขภาพ -> ประวัติวัคซีน -> คอก
// ต้องลบตารางลูกก่อน ไม่งั้นติด Foreign Key ถ้าขั้นไหนล้มจะ Rollback ทั้งหมด
// ถูกเรียกจาก: handlers/coop.go -> DeleteCoopHandler (DELETE /api/coops)
func DeleteCoop(coopID int, userID int) error {
	coop, err := GetCoopByIDForUser(coopID, userID)
	if err != nil {
		return err
	}

	// Use transaction to ensure all related records are deleted
	tx := DB.Begin()

	// Delete related records (cascading delete)
	if err := tx.Where("coop_id = ?", coopID).Delete(&models.Device{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting devices: %v", err)
		return err
	}

	if err := tx.Where("coop_id = ?", coopID).Delete(&models.Egg{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting eggs: %v", err)
		return err
	}

	if err := tx.Where("coop_id = ?", coopID).Delete(&models.Health{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting health records: %v", err)
		return err
	}

	if err := tx.Where("coop_id = ?", coopID).Delete(&models.VaccineHistory{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting vaccine history: %v", err)
		return err
	}

	// Delete the coop itself
	if err := tx.Delete(coop).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting coop: %v", err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	fmt.Printf("✓ Coop %d and all related data deleted\n", coopID)
	return nil
}
