// =====================================================================================
// ไฟล์ database/user_repository.go - บัญชีผู้ใช้ (ตาราง User)
// =====================================================================================
//
// ระบบไม่มีหน้าสมัครสมาชิก ผู้ดูแลเป็นคนสร้างบัญชีให้เจ้าของฟาร์มผ่าน API ที่ต้องใช้ X-Admin-Key
// รหัสผ่านเก็บเป็น hash ของ bcrypt เสมอ (ดู auth/password.go) ไม่เก็บรหัสผ่านจริง
// ถูกเรียกจาก: handlers/auth.go (login, register, ลบผู้ใช้, ดูข้อมูลตัวเอง)
package database

import (
	"errors"
	"log"

	"EZ-SmartFarm_BachN/models" // อย่าลืมเปลี่ยนตามชื่อโมดูลของคุณ

	"gorm.io/gorm"
)

// สมมติว่าตัวแปร DB ในโปรเจกต์ของคุณมีการประกาศเป็น *gorm.DB เอาไว้แล้ว
// เช่น var DB *gorm.DB

// GetUserByUsername ค้นหาผู้ใช้จาก username
//   - เจอ -> คืนข้อมูลผู้ใช้
//   - ไม่เจอ -> คืน (nil, nil) ไม่ใช่ error เพราะ "ไม่มีผู้ใช้นี้" เป็นกรณีปกติตอน login ผิด
//   - error อื่น (DB ล่ม) -> คืน error
//
// ข้อควรรู้: MySQL เทียบ username แบบไม่สนตัวพิมพ์เล็กใหญ่ ("Kkk" เจอ "kkk")
// LoginHandler จึงเทียบซ้ำใน Go อีกชั้นให้ต้องตรงทุกตัวอักษร
// ถูกเรียกจาก: handlers/auth.go -> LoginHandler
func GetUserByUsername(username string) (*models.User, error) {
	var user models.User

	// สั่งให้ GORM ค้นหาผู้ใช้ที่ username ตรงกัน และดึงข้อมูลบรรทัดแรก (First) มาใส่ในตัวแปร user
	result := DB.Where("username = ?", username).First(&user)

	if result.Error != nil {
		// ถ้า Error นั้นเกิดจากการ "ไม่พบข้อมูล"
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// ถ้าเกิด Error อื่นๆ (เช่น เน็ตหลุด, ฐานข้อมูลพัง)
		return nil, result.Error
	}

	return &user, nil
}

// เพิ่มฟังก์ชันนี้ต่อท้ายไฟล์ database/user_repository.go ที่มีอยู่เดิม

// CreateNewUser บันทึกผู้ใช้ใหม่ลงตาราง (ทำงานเหมือน CreateUser)
// เป็นฟังก์ชันเก่า ตอนนี้ไม่มีที่ไหนเรียกใช้ ใช้ CreateUser แทน
func CreateNewUser(user *models.User) error {
	// GORM จะทำการ INSERT ข้อมูลลงตารางให้ทันที
	result := DB.Create(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// CreateUser บันทึกผู้ใช้ใหม่ (username + hash ของรหัสผ่าน) ลงตาราง User
// ถูกเรียกจาก: handlers/auth.go -> RegisterHandler (ใช้ได้เฉพาะผู้ถือ X-Admin-Key)
func CreateUser(user *models.User) error {
	return DB.Create(user).Error
}

// GetUserByID ค้นหาผู้ใช้จาก id (คอลัมน์ id_User) ไม่เจอคืน ErrNotFound
// ถูกเรียกจาก: handlers/auth.go -> MeHandler (GET /api/auth/me ดูข้อมูลของตัวเองจาก JWT)
func GetUserByID(id int) (*models.User, error) {
	var user models.User
	if err := DB.Where("id_User = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UsernameExists เช็กว่า username นี้มีคนใช้แล้วหรือยัง (นับจำนวนแถวที่ชื่อตรงกัน)
// ถูกเรียกจาก: handlers/auth.go -> RegisterHandler ก่อนสร้างบัญชี เพื่อกันชื่อซ้ำ
func UsernameExists(username string) (bool, error) {
	var count int64
	if err := DB.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteUser ลบบัญชีผู้ใช้ "และข้อมูลทุกอย่างของผู้ใช้คนนั้น" แบบถาวร
//
// ใช้ Transaction (DB.Begin ... Commit): ถ้าลบขั้นไหนล้ม จะ Rollback ยกเลิกทั้งหมด
// ไม่มีกรณีลบไปครึ่งเดียว
//
// ลำดับการลบ (ต้องลบ "ลูก" ก่อน "แม่" ไม่งั้นติด Foreign Key):
//  1. หา coop_id ทุกคอกของผู้ใช้ (Pluck = ดึงคอลัมน์เดียวมาเป็น slice)
//  2. ลบข้อมูลที่ผูกกับคอก: ค่าเซนเซอร์ -> นัดตรวจ -> อุปกรณ์ -> ไข่ -> ผลตรวจสุขภาพ -> ประวัติวัคซีน -> คอก
//  3. ลบข้อมูลที่ผูกกับผู้ใช้โดยตรง: ค่ามาตรฐาน, กฎอาหารตามอายุ, ผังฟาร์ม, คลังอาหาร,
//     ล็อตนำเข้า, ประวัติแจกจ่าย, ชนิดอุปกรณ์, ประเภทวัคซีน
//  4. ลบแถวผู้ใช้
//  5. Commit ยืนยันทั้งหมด
//
// ถูกเรียกจาก: handlers/auth.go -> DeleteUserHandler (DELETE /api/auth/users ต้องใช้ X-Admin-Key)
func DeleteUser(userID int) error {
	tx := DB.Begin()

	var coopIDs []int
	if err := tx.Model(&models.Coop{}).Where("user_id = ?", userID).Pluck("coop_id", &coopIDs).Error; err != nil {
		tx.Rollback()
		log.Printf("Error listing coops for user %d: %v", userID, err)
		return err
	}

	if len(coopIDs) > 0 {
		// everything that hangs off a coop the user owns
		if err := tx.Where("device_id IN (?)", tx.Model(&models.Device{}).Select("device_id").Where("coop_id IN ?", coopIDs)).
			Delete(&models.SensorLog{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting sensor logs for user %d: %v", userID, err)
			return err
		}
		if err := tx.Where("coop_id IN ?", coopIDs).Delete(&models.HealthAppointment{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting health appointments for user %d: %v", userID, err)
			return err
		}
		if err := tx.Where("coop_id IN ?", coopIDs).Delete(&models.Device{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting devices for user %d: %v", userID, err)
			return err
		}
		if err := tx.Where("coop_id IN ?", coopIDs).Delete(&models.Egg{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting eggs for user %d: %v", userID, err)
			return err
		}
		if err := tx.Where("coop_id IN ?", coopIDs).Delete(&models.Health{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting health records for user %d: %v", userID, err)
			return err
		}
		if err := tx.Where("coop_id IN ?", coopIDs).Delete(&models.VaccineHistory{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting vaccine history for user %d: %v", userID, err)
			return err
		}
		if err := tx.Where("coop_id IN ?", coopIDs).Delete(&models.Coop{}).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting coops for user %d: %v", userID, err)
			return err
		}
	}

	// everything owned by the user directly (not through a coop)
	// device_type และ vaccine เป็นตารางที่ผูก user_id โดยตรงเหมือนกัน (เพิ่มทีหลัง
	// ตอนทำให้ชนิดอุปกรณ์/ชนิดวัคซีนแยกต่อผู้ใช้ - ลืมเพิ่มเข้าลิสต์นี้ตอนนั้น ทำให้
	// ลบ user ที่มีแถวในสองตารางนี้ไม่ได้เลย เจอ FK constraint ชนตอนลบ User ท้ายฟังก์ชัน)
	userScoped := []interface{}{
		&models.FarmThreshold{}, &models.FoodAgeRule{}, &models.FarmLayout{}, &models.Foodstock{},
		&models.ImportFood{}, &models.FoodDistribution{},
		&models.DeviceType{}, &models.Vaccine{},
	}
	for _, model := range userScoped {
		if err := tx.Where("user_id = ?", userID).Delete(model).Error; err != nil {
			tx.Rollback()
			log.Printf("Error deleting %T for user %d: %v", model, userID, err)
			return err
		}
	}

	if err := tx.Where("id_User = ?", userID).Delete(&models.User{}).Error; err != nil {
		tx.Rollback()
		log.Printf("Error deleting user %d: %v", userID, err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing user deletion for %d: %v", userID, err)
		return err
	}

	log.Printf("✓ Deleted user %d and all owned data (%d coop(s))", userID, len(coopIDs))
	return nil
}
