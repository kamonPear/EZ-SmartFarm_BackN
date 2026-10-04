package database

import (
	"errors"
	"log"

	"EZ-SmartFarm_BachN/models" // อย่าลืมเปลี่ยนตามชื่อโมดูลของคุณ

	"gorm.io/gorm"
)

// สมมติว่าตัวแปร DB ในโปรเจกต์ของคุณมีการประกาศเป็น *gorm.DB เอาไว้แล้ว
// เช่น var DB *gorm.DB

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

// ฟังก์ชันสร้างผู้ใช้ใหม่ลงตาราง User
func CreateNewUser(user *models.User) error {
	// GORM จะทำการ INSERT ข้อมูลลงตารางให้ทันที
	result := DB.Create(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// CreateUser creates a new user with the given username and password hash.
// Used by the admin-only register endpoint.
func CreateUser(user *models.User) error {
	return DB.Create(user).Error
}

// GetUserByID retrieves a user by their primary key id, or ErrNotFound if none exists.
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

// UsernameExists reports whether a user with the given username already exists.
func UsernameExists(username string) (bool, error) {
	var count int64
	if err := DB.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteUser permanently removes userID's account and every row owned by them,
// directly or through a coop they own - nothing is left behind. All in one
// transaction so a failure partway through leaves nothing deleted rather than a
// half-wiped account. Does NOT touch the `vaccine` table (types are shared/global
// across every user, not owned by any one of them - see models/vaccine.go).
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
	userScoped := []interface{}{
		&models.FarmThreshold{}, &models.FarmLayout{}, &models.Foodstock{},
		&models.ImportFood{}, &models.FoodDistribution{},
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
