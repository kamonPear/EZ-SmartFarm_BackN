// =====================================================================================
// ไฟล์ database/farm_layout_repository.go - อ่าน/บันทึก "รูปทรงผังฟาร์ม" ของผู้ใช้
// =====================================================================================
//
// ผังฟาร์มมี 3 รูปทรงให้เลือก: วงกลม / สามเหลี่ยม / สี่เหลี่ยม (หน้าจัดวางผังฟาร์มบนเว็บ)
// ผู้ใช้ 1 คนมีผังได้ 1 แถวในตาราง farm_layouts (ส่วนตำแหน่งคอกแต่ละคอกเก็บใน coop.pos_x/pos_y)
// ถูกเรียกจาก: handlers/farm_layout.go
package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFarmLayout ดึงรูปทรงผังฟาร์มของผู้ใช้ (userID)
//
//   - ถ้ามีแถวอยู่แล้ว -> คืนแถวนั้น
//   - ถ้า error อื่นที่ไม่ใช่ "ไม่พบข้อมูล" -> คืน error
//   - ถ้ายังไม่เคยเลือกรูปทรง -> คืนผังที่ Shape ว่าง "" โดย "ไม่บันทึกลง DB"
//
// ทำไมไม่สร้างแถวค่าเริ่มต้นให้เลย: เดิมเคยทำ ผลคือแค่เปิดหน้าแรกก็ได้ผัง "วงกลม" ทั้งที่ผู้ใช้
// ไม่เคยเลือก หน้าบ้านจึงแยกไม่ออกว่า "ยังไม่ได้ตั้ง" หรือ "ตั้งเป็นวงกลม"
// แถวจะถูกสร้างเฉพาะใน UpdateFarmLayoutShape ตอนผู้ใช้กดบันทึกจริงเท่านั้น
// ถูกเรียกจาก: handlers/farm_layout.go -> GetFarmLayoutHandler
func GetFarmLayout(userID int) (*models.FarmLayout, error) {
	var layout models.FarmLayout

	err := DB.Where("user_id = ?", userID).First(&layout).Error
	if err == nil {
		return &layout, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("Error fetching farm layout: %v", err)
		return nil, err
	}

	return &models.FarmLayout{UserID: userID, Shape: ""}, nil
}

// UpdateFarmLayoutShape บันทึกรูปทรงผังที่ผู้ใช้เลือก (circle/triangle/square)
//
// ขั้นตอน:
//  1. หาแถวของผู้ใช้ ถ้ายังไม่มี (ใช้ครั้งแรก) สร้างแถวใหม่ก่อน
//  2. ใส่ Shape ใหม่แล้ว DB.Save (Save = อัปเดตทุกคอลัมน์ของแถวนั้น)
//
// ถูกเรียกจาก: handlers/farm_layout.go -> UpdateFarmLayoutHandler (ปุ่ม "บันทึกผังฟาร์ม" บนเว็บ)
func UpdateFarmLayoutShape(shape string, userID int) (*models.FarmLayout, error) {
	var layout models.FarmLayout
	err := DB.Where("user_id = ?", userID).First(&layout).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("Error fetching farm layout: %v", err)
			return nil, err
		}
		layout = models.FarmLayout{UserID: userID}
		if err := DB.Create(&layout).Error; err != nil {
			log.Printf("Error creating farm layout: %v", err)
			return nil, err
		}
	}

	layout.Shape = shape
	if err := DB.Save(&layout).Error; err != nil {
		log.Printf("Error updating farm layout shape: %v", err)
		return nil, err
	}

	log.Printf("✓ Farm layout shape set to %s for user %d\n", shape, userID)
	return &layout, nil
}
