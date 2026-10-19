// =====================================================================================
// ไฟล์ database/farm_threshold_repository.go - ค่ามาตรฐานของฟาร์ม (อุณหภูมิ / แอมโมเนีย)
// =====================================================================================
//
// ค่ามาตรฐานคือค่าที่เจ้าของฟาร์มตั้งไว้ใช้ร่วมกันทุกคอก เช่น 25°C และ 20 ppm
// ใช้เทียบกับค่าที่เซนเซอร์วัดได้ และส่งต่อไปให้บอร์ด ESP32 ผ่าน MQTT (services/threshold_publisher.go)
//
// เดิมเว็บเก็บค่าใน localStorage และแอปเก็บใน SharedPreferences แยกกัน ทำให้สองฝั่งแสดงค่าไม่ตรงกัน
// จึงย้ายมาเก็บในตาราง farm_thresholds (1 แถวต่อผู้ใช้ 1 คน) เว็บกับแอปจึงเห็นค่าเดียวกันเสมอ
package database

import (
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// GetFarmThreshold ดึงค่ามาตรฐานของผู้ใช้
//
//   - มีแถวแล้ว -> คืนค่าที่ตั้งไว้
//   - ยังไม่เคยตั้ง -> คืนค่า 0/0 โดยไม่บันทึกลง DB
//
// ทำไมคืน 0 และไม่ใส่ค่าเริ่มต้นให้: "ยังไม่ได้ตั้งค่า" ต้องเป็นสถานะที่มองเห็นได้จริง
// หน้าหลักจะแสดงข้อความ "แตะเพื่อตั้งค่า" แทนการโชว์ตัวเลขปลอมที่ผู้ใช้ไม่ได้ตั้งเอง
// ถูกเรียกจาก: handlers/farm_threshold.go (GET /api/farm-threshold) และ services/threshold_publisher.go (ส่งค่าให้บอร์ด)
func GetFarmThreshold(userID int) (*models.FarmThreshold, error) {
	var threshold models.FarmThreshold

	err := DB.Where("user_id = ?", userID).First(&threshold).Error
	if err == nil {
		return &threshold, nil
	}

	if err != gorm.ErrRecordNotFound {
		log.Printf("Error fetching farm threshold: %v", err)
		return nil, err
	}

	return &models.FarmThreshold{UserID: userID, Temperature: 0, Ammonia: 0}, nil
}

// UpdateFarmThreshold บันทึกค่าอุณหภูมิและแอมโมเนียที่ผู้ใช้ตั้ง
// ถ้าเป็นครั้งแรกจะสร้างแถวใหม่ก่อน แล้วอัปเดตเป็นค่าที่ส่งมา (req.Temperature, req.Ammonia)
// ถูกเรียกจาก: handlers/farm_threshold.go -> UpdateFarmThresholdHandler (ปุ่ม "บันทึกค่ามาตรฐาน")
func UpdateFarmThreshold(req *models.UpdateFarmThresholdRequest, userID int) (*models.FarmThreshold, error) {
	var threshold models.FarmThreshold
	err := DB.Where("user_id = ?", userID).First(&threshold).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("Error fetching farm threshold: %v", err)
			return nil, err
		}
		threshold = models.FarmThreshold{UserID: userID}
		if err := DB.Create(&threshold).Error; err != nil {
			log.Printf("Error creating farm threshold: %v", err)
			return nil, err
		}
	}

	threshold.Temperature = req.Temperature
	threshold.Ammonia = req.Ammonia
	if err := DB.Save(&threshold).Error; err != nil {
		log.Printf("Error updating farm threshold: %v", err)
		return nil, err
	}

	log.Printf("✓ Farm threshold set to %.1f°C / %.1f ppm for user %d\n", req.Temperature, req.Ammonia, userID)
	return &threshold, nil
}
