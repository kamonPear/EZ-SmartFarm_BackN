package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"
)

// CreateVaccine creates a new vaccine administration record for a coop.
// The coop must already be loaded (via GetCoopByID) so name_coop/birthday can be
// copied onto the vaccine row to satisfy the fk_name_coop_vaccines constraint.
func CreateVaccine(req *models.CreateVaccineRequest, coop *models.Coop) (*models.Vaccine, error) {
	vaccine := models.Vaccine{
		CoopID:         req.CoopID,
		NameCoop:       coop.NameCoop,
		Birthday:       &coop.Birthday,
		Name:           req.Name,
		RecordDate:     req.RecordDate,
		Method:         req.Method,
		RecommendedAge: req.RecommendedAge,
		Note:           req.Note,
	}

	if err := DB.Create(&vaccine).Error; err != nil {
		log.Printf("Error creating vaccine: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Vaccine created with ID: %d\n", vaccine.VaccineID)
	return &vaccine, nil
}

// GetVaccineByID retrieves a vaccine record by vaccine ID
func GetVaccineByID(vaccineID int) (*models.Vaccine, error) {
	var vaccine models.Vaccine

	result := DB.Where("vaccine_id = ?", vaccineID).First(&vaccine)
	if result.Error != nil {
		log.Printf("Error retrieving vaccine by ID %d: %v", vaccineID, result.Error)
		return nil, result.Error
	}

	return &vaccine, nil
}

// GetVaccinesByCoopID retrieves all vaccine records for a specific coop
func GetVaccinesByCoopID(coopID int) ([]models.Vaccine, error) {
	var vaccines []models.Vaccine

	result := DB.Where("coop_id = ?", coopID).Find(&vaccines)
	if result.Error != nil {
		log.Printf("Error retrieving vaccines for coop ID %d: %v", coopID, result.Error)
		return nil, result.Error
	}

	return vaccines, nil
}

// GetAllVaccines retrieves all vaccine records from database
func GetAllVaccines() ([]models.Vaccine, error) {
	var vaccines []models.Vaccine

	err := DB.Debug().Preload("Coop").Find(&vaccines).Error

	if err != nil {
		log.Printf("Error retrieving all vaccines: %v", err)
	}

	return vaccines, err
}

// DeleteVaccine deletes a vaccine record by vaccine ID
func DeleteVaccine(vaccineID int) error {
	result := DB.Where("vaccine_id = ?", vaccineID).Delete(&models.Vaccine{})
	if result.Error != nil {
		log.Printf("Error deleting vaccine ID %d: %v", vaccineID, result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("No vaccine found with ID %d", vaccineID)
		return fmt.Errorf("no vaccine found with ID %d", vaccineID)
	}

	log.Printf("Successfully deleted vaccine ID %d", vaccineID)
	return nil
}

// DeleteVaccinesByCoopID deletes all vaccine records for a specific coop
func DeleteVaccinesByCoopID(coopID int) error {
	result := DB.Where("coop_id = ?", coopID).Delete(&models.Vaccine{})
	if result.Error != nil {
		log.Printf("Error deleting vaccines for coop ID %d: %v", coopID, result.Error)
		return result.Error
	}

	log.Printf("Successfully deleted %d vaccine records for coop ID %d", result.RowsAffected, coopID)
	return nil
}

// ==========================================
// 🌟 ส่วนระบบแจ้งเตือน 🌟
// ==========================================

// PendingVaccineAlert โครงสร้างสำหรับส่งแจ้งเตือนไปให้ Flutter
type PendingVaccineAlert struct {
	Title string `json:"title"`
	Date  string `json:"date"`
	Time  string `json:"time"`
}

// GetPendingVaccinesAlerts ฟังก์ชันคำนวณวัคซีนที่ถึงกำหนด
func GetPendingVaccinesAlerts() ([]PendingVaccineAlert, error) {
	var coops []models.Coop
	var schedules []models.MedicineSchedule
	var alerts []PendingVaccineAlert

	// 1. ดึงข้อมูลคอกไก่ทั้งหมด
	if err := DB.Find(&coops).Error; err != nil {
		return nil, err
	}

	// 2. ดึงตารางเกณฑ์วัคซีนทั้งหมด
	if err := DB.Find(&schedules).Error; err != nil {
		return nil, err
	}

	// 3. เตรียมเวลาปัจจุบัน
	now := time.Now()
	todayDate := now.Format("02/01/") + fmt.Sprintf("%d", now.Year()+543)
	timeNow := now.Format("15:04")

	// 4. เทียบอายุไก่กับเกณฑ์
	for _, coop := range coops {
		// ✅ เช็คว่ามีการระบุ วันเกิด และ วันที่นำเข้าเลี้ยง (DateAdoptAnimals)
		if !coop.Birthday.IsZero() && !coop.DateAdoptAnimals.IsZero() {

			// 🛑 ถ้าปัจจุบันยังไม่ถึงวันที่นำไก่เข้าเลี้ยง -> ยังไม่ต้องเริ่มให้วัคซีน ข้ามไปเลย
			if now.Before(coop.DateAdoptAnimals) {
				continue
			}

			// คำนวณอายุไก่ปัจจุบัน (เทียบจากวันเกิด)
			ageInDays := models.CalculateChickenAge(coop.Birthday)

			// คำนวณอายุไก่ "ณ วันที่ถูกนำเข้าเลี้ยง" (ระยะห่างระหว่างวันเกิดถึง DateAdoptAnimals)
			ageAtImport := int(coop.DateAdoptAnimals.Sub(coop.Birthday).Hours() / 24)

			for _, schedule := range schedules {
				// 🛑 เช็คว่าวัคซีนตัวนี้ เลยกำหนดไปหรือยังตั้งแต่ก่อนไก่เข้าเลี้ยง
				// ตัวอย่าง: ไก่เข้าวันที่ 12 (ageAtImport=12) แต่วัคซีนให้ตอนอายุไม่เกิน 7 วัน (MaxAgeDays=7) -> ข้าม
				if schedule.MaxAgeDays < ageAtImport {
					continue
				}

				// ✅ ถ้าอายุไก่ปัจจุบันอยู่ในช่วงที่ต้องให้วัคซีน
				if ageInDays >= schedule.MinAgeDays && ageInDays <= schedule.MaxAgeDays {

					alertTitle := fmt.Sprintf("⚠️ ถึงกำหนดให้ %s ที่คอก %d (อายุ %d วัน)", schedule.Name, coop.CoopID, ageInDays)

					alerts = append(alerts, PendingVaccineAlert{
						Title: alertTitle,
						Date:  todayDate,
						Time:  timeNow,
					})
				}
			}
		}
	}

	return alerts, nil
}