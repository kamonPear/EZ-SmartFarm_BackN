// =====================================================================================
// ไฟล์ database/vaccine_repository.go - "ประวัติการให้วัคซีน" ของแต่ละคอก (ตาราง vaccine_history)
// =====================================================================================
//
// ระวังสับสน มี 2 ตารางเรื่องวัคซีน:
//   - vaccine         : "ประเภท" วัคซีน/ยาที่ผู้ใช้ตั้งไว้ (ชื่อ, วิธีให้, ช่วงอายุที่ให้) -> หน้า "เพิ่มประเภทวัคซีน/ยา"
//   - vaccine_history : บันทึกว่า "คอกไหนได้รับวัคซีนอะไร วันไหน" จริง <- ไฟล์นี้
//     -> สร้างเมื่อกด "ให้วัคซีนแล้ว" และแก้ไขได้ในหน้าแก้ไขข้อมูลคอกไก่
//
// ทุกฟังก์ชันเช็กว่าคอกเป็นของผู้ใช้ก่อนเสมอ ผ่าน subquery หรือ CoopBelongsToUser
// ถูกเรียกจาก: handlers/vaccine.go
package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// CreateVaccine บันทึกการให้วัคซีน 1 ครั้งของคอก
// รับข้อมูลคอก (coop) ที่ผู้เรียกโหลดและตรวจสิทธิ์มาแล้ว แล้วคัดลอกชื่อคอกและวันเกิดไก่ลงแถวประวัติด้วย
// ทำไมต้องคัดลอก: ให้ประวัติแสดงข้อมูล ณ ตอนที่ให้วัคซีน แม้ภายหลังจะแก้ชื่อคอก
// ถูกเรียกจาก: handlers/vaccine.go (ปุ่ม "ให้วัคซีนแล้ว" / "เสร็จสิ้น" ในหน้าแจ้งเตือน)
func CreateVaccine(req *models.CreateVaccineRequest, coop *models.Coop) (*models.VaccineHistory, error) {
	history := models.VaccineHistory{
		CoopID:         req.CoopID,
		NameCoop:       coop.NameCoop,
		Birthday:       coop.Birthday,
		Name:           req.Name,
		RecordDate:     req.RecordDate,
		Method:         req.Method,
		RecommendedAge: req.RecommendedAge,
		Note:           req.Note,
	}

	if err := DB.Create(&history).Error; err != nil {
		log.Printf("Error creating vaccine history record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Vaccine history record created with ID: %d\n", history.VaccineID)
	return &history, nil
}

// GetVaccineByID ดึงประวัติวัคซีน 1 รายการตาม id เฉพาะของคอกที่ผู้ใช้เป็นเจ้าของ
// ไม่เจอ หรือเป็นของคนอื่น -> ErrNotFound
// ถูกใช้ใน UpdateVaccine / DeleteVaccine ด้านล่าง และ handlers/vaccine.go
func GetVaccineByID(vaccineID int, userID int) (*models.VaccineHistory, error) {
	var history models.VaccineHistory

	result := DB.Where("id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", vaccineID, userID).First(&history)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error retrieving vaccine history by ID %d: %v", vaccineID, result.Error)
		return nil, result.Error
	}

	return &history, nil
}

// GetVaccinesByCoopID ดึงประวัติวัคซีนทั้งหมดของคอกเดียว (หน้าต่าง "ดูประวัติ" ในหน้าให้วัคซีน)
// เช็กก่อนว่าคอกเป็นของผู้ใช้ ถ้าไม่ใช่คืน ErrNotFound
func GetVaccinesByCoopID(coopID int, userID int) ([]models.VaccineHistory, error) {
	var history []models.VaccineHistory

	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	result := DB.Where("coop_id = ?", coopID).Find(&history)
	if result.Error != nil {
		log.Printf("Error retrieving vaccine history for coop ID %d: %v", coopID, result.Error)
		return nil, result.Error
	}

	return history, nil
}

// GetAllVaccines ดึงประวัติวัคซีนของทุกคอกของผู้ใช้ พร้อมข้อมูลคอก (Preload)
// ถูกเรียกจาก: handlers/vaccine.go (ใช้ทั้งแสดงผล และเช็กว่าวัคซีนไหนให้ไปแล้ว)
func GetAllVaccines(userID int) ([]models.VaccineHistory, error) {
	var history []models.VaccineHistory

	err := DB.Preload("Coop").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Find(&history).Error

	if err != nil {
		log.Printf("Error retrieving all vaccine history: %v", err)
	}

	return history, err
}

// UpdateVaccine แก้ไขประวัติวัคซีน (หน้าต่าง "แก้ไข บันทึกวัคซีน")
// แก้เฉพาะช่องที่ส่งค่ามา ช่องที่ว่าง ("" หรือวันที่ศูนย์) จะคงค่าเดิมไว้
// เช็กสิทธิ์ผ่าน GetVaccineByID ก่อน แล้วค่อย DB.Save
func UpdateVaccine(id int, req *models.UpdateVaccineRequest, userID int) (*models.VaccineHistory, error) {
	history, err := GetVaccineByID(id, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		history.Name = req.Name
	}
	if req.Method != "" {
		history.Method = req.Method
	}
	if req.RecommendedAge != "" {
		history.RecommendedAge = req.RecommendedAge
	}
	if !req.RecordDate.IsZero() {
		history.RecordDate = req.RecordDate
	}
	if req.Note != "" {
		history.Note = req.Note
	}

	if err := DB.Save(history).Error; err != nil {
		log.Printf("Error updating vaccine history ID %d: %v", id, err)
		return nil, err
	}

	return history, nil
}

// DeleteVaccine ลบประวัติวัคซีน 1 รายการ (เช็กสิทธิ์ผ่าน GetVaccineByID ก่อน)
func DeleteVaccine(vaccineID int, userID int) error {
	history, err := GetVaccineByID(vaccineID, userID)
	if err != nil {
		return err
	}

	result := DB.Delete(history)
	if result.Error != nil {
		log.Printf("Error deleting vaccine history ID %d: %v", vaccineID, result.Error)
		return result.Error
	}

	log.Printf("Successfully deleted vaccine history ID %d", vaccineID)
	return nil
}

// DeleteVaccinesByCoopID ลบประวัติวัคซีนทั้งหมดของคอก "โดยไม่เช็กเจ้าของ"
// ห้ามเรียกตรงจาก handler ให้เรียกผ่าน DeleteVaccinesByCoopIDForUser ที่เช็กสิทธิ์ก่อน
func DeleteVaccinesByCoopID(coopID int) error {
	result := DB.Where("coop_id = ?", coopID).Delete(&models.VaccineHistory{})
	if result.Error != nil {
		log.Printf("Error deleting vaccine history for coop ID %d: %v", coopID, result.Error)
		return result.Error
	}

	log.Printf("Successfully deleted %d vaccine history records for coop ID %d", result.RowsAffected, coopID)
	return nil
}

// DeleteVaccinesByCoopIDForUser ลบประวัติวัคซีนทั้งหมดของคอก หลังเช็กว่าคอกเป็นของผู้ใช้แล้ว
// ถูกเรียกจาก: handlers/vaccine.go
func DeleteVaccinesByCoopIDForUser(coopID int, userID int) error {
	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	return DeleteVaccinesByCoopID(coopID)
}
