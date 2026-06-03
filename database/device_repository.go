package database

import (
	"time"
	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"// สมมติว่าใช้ gorm v1 หรือ v2 ปรับตามอิมพอร์ตของคุณนะครับ
)

// SaveCoopLayout บันทึกพิกัดอุปกรณ์แบบ GORM Style
func SaveCoopLayout(coopID int, slots []models.SlotPayload) error {
	// ใช้ GORM Transaction อัตโนมัติ (ถ้าด้านใน return error มันจะ Rollback ให้เอง)
	return DB.Transaction(func(tx *gorm.DB) error {
		
		// 1. ลบอุปกรณ์เดิมในเล้าเพื่อเคลียร์ Layout เก่า
		if err := tx.Where("coop_id = ?", coopID).Delete(&models.Device{}).Error; err != nil {
			return err
		}

		// 2. วนลูปบันทึกเฉพาะช่องที่กดวางอุปกรณ์ (ช่องที่มี device ไม่เป็น nil)
		for _, slot := range slots {
			if slot.Device != nil {
				deviceType := "Sensor"
				if slot.Device.Name == "พัดลม" || slot.Device.Name == "หลอดไฟ" {
					deviceType = "Actuator"
				}

				// สร้าง Object จาก Model Device ของคุณ
				newDevice := models.Device{
					CoopID:        coopID,
					SlotIndex:     slot.ID, // ไอดีช่อง 0-20 จาก Angular
					Name:          slot.Device.Name,
					Icon:          slot.Device.Icon,
					DeviceType:    deviceType,
					CurrentStatus: "Offline",
					LastUpdate:    time.Now(),
				}

				// สั่ง GORM บันทึกลงตาราง device
				if err := tx.Create(&newDevice).Error; err != nil {
					return err
				}
			}
		}
		
		return nil
	})
}