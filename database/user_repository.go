package database

import (
	"errors"
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
