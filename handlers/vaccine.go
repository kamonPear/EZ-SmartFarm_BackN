// =====================================================================================
// ไฟล์ handlers/vaccine.go - API วัคซีน/ยา ทั้งหมด (ไฟล์ใหญ่ที่สุดของ handlers)
// =====================================================================================
//
// มี 2 เรื่องที่ต้องแยกให้ออก:
//
//	A) "ประเภท" วัคซีน/ยา (ตาราง vaccine) = เกณฑ์ที่ผู้ใช้ตั้ง: ชื่อ, วิธีให้, ช่วงอายุ min_age_days - max_age_days
//	   API: /api/vaccines/schedule ...  (หน้า "เพิ่มประเภทวัคซีน/ยา")
//	B) "ประวัติการให้จริง" (ตาราง vaccine_history) = คอกไหนได้วัคซีนอะไร วันไหน
//	   API: /api/vaccines  (ข้อมูลอยู่ใน database/vaccine_repository.go)
//
// หลักคิดวันครบกำหนดให้วัคซีน (ใช้ซ้ำหลายฟังก์ชัน และใน calendar.go):
//
//	วันครบกำหนด = วันเกิดไก่ + min_age_days
//	- ถ้าช่วงอายุจบ (วันเกิด + max_age_days) ก่อนวันรับเข้าเลี้ยง -> ไก่โตเกินช่วงก่อนมาถึงฟาร์ม
//	  (น่าจะได้รับจากที่อื่นแล้ว) ไม่ต้องแจ้งเตือน
//	- ถ้าวันครบกำหนดอยู่ก่อนวันรับเข้าเลี้ยง -> เลื่อนเป็นวันรับเข้าเลี้ยง (ให้ได้ทันทีที่รับมา)
//
// "ให้แล้วหรือยัง" ดูจากว่ามีแถวใน vaccine_history ที่ (coop_id, ชื่อวัคซีน) ตรงกันไหม
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateVaccineHandler (POST /api/vaccines) บันทึกประวัติการให้วัคซีน 1 ครั้ง
// ต้องมี coop_id, ชื่อ, วิธีให้, วันที่ให้ ไม่งั้น 400
// โหลดคอกผ่าน GetCoopByIDForUser (เช็กเจ้าของไปด้วย) ไม่ใช่ของผู้ใช้ -> 404
// แล้วส่งคอกให้ database.CreateVaccine คัดลอกชื่อคอก/วันเกิดไก่ลงประวัติ สำเร็จ -> 201
func CreateVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateVaccineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CoopID == 0 || req.Name == "" || req.Method == "" || req.RecordDate.IsZero() {
		log.Printf("[%s] %s - %d (Missing required fields)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing required fields: coop_id, name, method, record_date", http.StatusBadRequest)
		return
	}

	coop, err := database.GetCoopByIDForUser(req.CoopID, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Coop not found: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
		http.Error(w, "Coop not found", http.StatusNotFound)
		return
	}

	vaccine, err := database.CreateVaccine(&req, coop)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to create vaccine: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to create vaccine", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	log.Printf("[%s] %s - %d ✓ Created vaccine ID: %d", r.Method, r.RequestURI, http.StatusCreated, vaccine.VaccineID)
	json.NewEncoder(w).Encode(vaccine)
}

// UpdateVaccineHandler (PUT /api/vaccines?id=) แก้ไขประวัติการให้วัคซีน
// (หน้าต่าง "แก้ไข บันทึกวัคซีน" ในหน้าแก้ไขข้อมูลคอก) ไม่เจอ/ไม่ใช่ของผู้ใช้ -> 404
func UpdateVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing vaccine id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing vaccine id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid vaccine id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid vaccine id", http.StatusBadRequest)
		return
	}

	var req models.UpdateVaccineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	vaccine, err := database.UpdateVaccine(id, &req, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to update vaccine: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
		http.Error(w, "Vaccine not found", http.StatusNotFound)
		return
	}

	log.Printf("[%s] %s - %d ✓ Updated vaccine ID: %d", r.Method, r.RequestURI, http.StatusOK, vaccine.VaccineID)
	json.NewEncoder(w).Encode(vaccine)
}

// GetVaccineHandler (GET /api/vaccines) ดึงประวัติการให้วัคซีน รองรับ 3 แบบ:
//   - ?id=      : รายการเดียว
//   - ?coop_id= : ทั้งหมดของคอกนั้น (หน้าต่าง "ดูประวัติ")
//   - ไม่ใส่    : ทุกคอกของผู้ใช้
func GetVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	vaccineID := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	// Get vaccine by ID
	if vaccineID != "" {
		id, err := strconv.Atoi(vaccineID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid vaccine ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid vaccine ID format", http.StatusBadRequest)
			return
		}

		vaccine, err := database.GetVaccineByID(id, userID)
		if err != nil {
			log.Printf("[%s] %s - %d (Vaccine not found: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
			http.Error(w, "Vaccine not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Retrieved vaccine ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
		json.NewEncoder(w).Encode(vaccine)
		return
	}

	// Get vaccines by coop ID
	if coopID != "" {
		id, err := strconv.Atoi(coopID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid coop ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		vaccines, err := database.GetVaccinesByCoopID(id, userID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
				http.Error(w, "Coop not found", http.StatusNotFound)
				return
			}
			log.Printf("[%s] %s - %d (Failed to retrieve vaccines: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
			http.Error(w, "Failed to retrieve vaccines", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Retrieved %d vaccines for coop ID: %d", r.Method, r.RequestURI, http.StatusOK, len(vaccines), id)
		json.NewEncoder(w).Encode(vaccines)
		return
	}

	// Get all vaccines
	vaccines, err := database.GetAllVaccines(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve vaccines: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve vaccines", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved %d vaccines", r.Method, r.RequestURI, http.StatusOK, len(vaccines))
	json.NewEncoder(w).Encode(vaccines)
}

// DeleteVaccineHandler (DELETE /api/vaccines) ลบประวัติการให้วัคซีน รองรับ 2 แบบ:
//   - ?id=      : ลบรายการเดียว
//   - ?coop_id= : ลบทั้งหมดของคอกนั้น (DeleteVaccinesByCoopIDForUser เช็กเจ้าของก่อน)
//     ไม่ใส่ทั้งคู่ -> 400
func DeleteVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	vaccineID := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	// Delete vaccine by ID
	if vaccineID != "" {
		id, err := strconv.Atoi(vaccineID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid vaccine ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid vaccine ID format", http.StatusBadRequest)
			return
		}

		err = database.DeleteVaccine(id, userID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				log.Printf("[%s] %s - %d (Vaccine not found)", r.Method, r.RequestURI, http.StatusNotFound)
				http.Error(w, "Vaccine not found", http.StatusNotFound)
				return
			}
			log.Printf("[%s] %s - %d (Failed to delete vaccine: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
			http.Error(w, "Failed to delete vaccine", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Deleted vaccine ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
		json.NewEncoder(w).Encode(map[string]string{"message": "Vaccine deleted successfully"})
		return
	}

	// Delete vaccines by coop ID
	if coopID != "" {
		id, err := strconv.Atoi(coopID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid coop ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		err = database.DeleteVaccinesByCoopIDForUser(id, userID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
				http.Error(w, "Coop not found", http.StatusNotFound)
				return
			}
			log.Printf("[%s] %s - %d (Failed to delete vaccines: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
			http.Error(w, "Failed to delete vaccines", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Deleted all vaccines for coop ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
		json.NewEncoder(w).Encode(map[string]string{"message": "All vaccines for coop deleted successfully"})
		return
	}

	log.Printf("[%s] %s - %d (Missing vaccine_id or coop_id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
	http.Error(w, "Missing vaccine_id or coop_id parameter", http.StatusBadRequest)
}

// scheduleRequest JSON ที่แอป/เว็บส่งมาตอนเพิ่ม/แก้ไข "ประเภท" วัคซีน/ยา
// แยกจาก models.Vaccine เพื่อคงชื่อ field เดิมที่ฝั่งแอปใช้อยู่ (เช่น description แทน note)
// แม้ข้างในฐานข้อมูลจะเปลี่ยนโครงสร้างไปแล้ว แอปก็ไม่ต้องแก้
type scheduleRequest struct {
	Name        string `json:"name"`
	Method      string `json:"method"`
	MinAgeDays  int    `json:"min_age_days"`
	MaxAgeDays  int    `json:"max_age_days"`
	Description string `json:"description"`
}

// scheduleView รูปแบบข้อมูล "ประเภท" วัคซีนที่ส่งกลับให้แอป/เว็บ
// หน้าตาเหมือนตอนที่ยังเป็นตาราง medicine_schedules แยกในรุ่นเก่า ฝั่งแอปจึงไม่ต้องแก้
type scheduleView struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Method      string `json:"method"`
	MinAgeDays  int    `json:"min_age_days"`
	MaxAgeDays  int    `json:"max_age_days"`
	Description string `json:"description"`
}

// vaccineToScheduleView แปลงแถว models.Vaccine จาก DB เป็น scheduleView ที่ส่งให้แอป
// (เช่น Note ใน DB -> description ใน JSON)
func vaccineToScheduleView(v models.Vaccine) scheduleView {
	return scheduleView{
		ID:          v.VaccineID,
		Name:        v.Name,
		Method:      v.Method,
		MinAgeDays:  v.MinAgeDays,
		MaxAgeDays:  v.MaxAgeDays,
		Description: v.Note,
	}
}

// AddCustomMedicineHandler (POST /api/vaccines/schedule) เพิ่ม "ประเภท" วัคซีน/ยาใหม่
// (หน้า "เพิ่มประเภทวัคซีน/ยา" ชื่อ, วิธีให้, ช่วงอายุ, คำอธิบาย)
//  1. กันชื่อซ้ำในฟาร์มเดียวกัน แบบไม่สนตัวพิมพ์ ซ้ำ -> 409 พร้อมข้อความไทย
//     ทำไมต้องกันซ้ำ: ระบบจับคู่ประวัติกับประเภทด้วย "ชื่อ" ถ้าชื่อซ้ำจะแยกไม่ออก
//  2. บันทึกลงตาราง vaccine (ผูก user_id) ตอบ 201
func AddCustomMedicineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 🛑 กันชื่อยา/วัคซีนซ้ำ (ไม่สนตัวพิมพ์เล็ก-ใหญ่) - เฉพาะในฟาร์ม (user) เดียวกัน
	// เท่านั้น ฟาร์มอื่นตั้งชื่อซ้ำกันได้ เพราะเป็นคนละฟาร์มกัน
	var existingCount int64
	if err := database.DB.Model(&models.Vaccine{}).
		Where("user_id = ? AND LOWER(name_vaccine) = LOWER(?)", userID, req.Name).
		Count(&existingCount).Error; err != nil {
		log.Printf("Failed to check duplicate medicine name: %v", err)
		http.Error(w, "Failed to save medicine", http.StatusInternalServerError)
		return
	}
	if existingCount > 0 {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "มียา/วัคซีนชื่อนี้อยู่ในระบบแล้ว กรุณาใช้ชื่ออื่น",
		})
		return
	}

	vaccine := models.Vaccine{
		UserID:     userID,
		Name:       req.Name,
		Method:     req.Method,
		Note:       req.Description,
		MinAgeDays: req.MinAgeDays,
		MaxAgeDays: req.MaxAgeDays,
	}
	if err := database.DB.Create(&vaccine).Error; err != nil {
		log.Printf("Failed to save medicine: %v", err)
		http.Error(w, "Failed to save medicine", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "บันทึกข้อมูลยา/วัคซีนสำเร็จ",
		"data":    vaccineToScheduleView(vaccine),
	})
}

// GetMedicineSchedulesHandler (GET /api/vaccines/schedule) ดึงประเภทวัคซีน/ยาทั้งหมดของผู้ใช้
// เรียงตามชื่อ ใช้แสดงรายการในหน้าวัคซีน
func GetMedicineSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var rows []models.Vaccine
	if err := database.DB.Where("user_id = ?", userID).Order("name_vaccine asc").Find(&rows).Error; err != nil {
		log.Printf("Failed to fetch medicine schedules: %v", err)
		http.Error(w, "Failed to fetch medicine schedules", http.StatusInternalServerError)
		return
	}

	schedules := make([]scheduleView, 0, len(rows))
	for _, row := range rows {
		schedules = append(schedules, vaccineToScheduleView(row))
	}
	json.NewEncoder(w).Encode(schedules)
}

// GetRecommendedVaccinesHandler (GET /api/vaccines/recommended?coop_id=) แนะนำวัคซีนตามอายุไก่
// มี coop_id:
//  1. โหลดคอก (เช็กเจ้าของ) คำนวณอายุไก่เป็นวัน (models.CalculateChickenAge)
//  2. ถ้ายังไม่ถึงวันรับเข้าเลี้ยง -> ส่งรายการว่าง (ยังไม่มีไก่ในฟาร์ม)
//  3. หาประเภทที่ min_age_days <= อายุ <= max_age_days
//
// ไม่มี coop_id: ส่งประเภททั้งหมดของผู้ใช้กลับไป
func GetRecommendedVaccinesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	coopID := r.URL.Query().Get("coop_id")

	if coopID != "" {
		id, err := strconv.Atoi(coopID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid coop ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		coop, err := database.GetCoopByIDForUser(id, userID)
		if err != nil {
			log.Printf("[%s] %s - %d (Coop not found: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}

		ageInDays := models.CalculateChickenAge(coop.Birthday)
		log.Printf("DEBUG: Coop ID %d - Birthday: %v, Current Age: %d days", id, coop.Birthday, ageInDays)

		var recommendedRows []models.Vaccine

		// 🌟 ตรวจสอบว่าถึงวันที่นำไก่เข้าหรือยัง (เทียบแบบ YYYY-MM-DD ตัดปัญหาเรื่องเวลา)
		now := time.Now()
		if !coop.DateAdoptAnimals.IsZero() && models.DateKey(now) < models.DateKey(coop.DateAdoptAnimals) {
			// ถ้ายังไม่ถึงวันนำไก่เข้า ให้ส่งอาเรย์ว่างกลับไปเลย เพราะยังไม่ต้องแนะนำ
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"coop_id":              id,
				"birthday":             coop.Birthday,
				"current_age_days":     ageInDays,
				"recommended_vaccines": []scheduleView{},
			})
			return
		}

		err = database.DB.Where("user_id = ? AND min_age_days <= ? AND max_age_days >= ?", userID, ageInDays, ageInDays).Find(&recommendedRows).Error
		if err != nil {
			log.Printf("Failed to fetch medicines: %v", err)
			http.Error(w, "Failed to fetch medicines", http.StatusInternalServerError)
			return
		}
		recommendedMedicines := make([]scheduleView, 0, len(recommendedRows))
		for _, row := range recommendedRows {
			recommendedMedicines = append(recommendedMedicines, vaccineToScheduleView(row))
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Retrieved recommended vaccines for coop ID: %d (age: %d days)", r.Method, r.RequestURI, http.StatusOK, id, ageInDays)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"coop_id":              id,
			"birthday":             coop.Birthday,
			"current_age_days":     ageInDays,
			"recommended_vaccines": recommendedMedicines,
		})
		return
	}

	var allRows []models.Vaccine
	if err := database.DB.Where("user_id = ?", userID).Find(&allRows).Error; err != nil {
		log.Printf("Failed to fetch all schedules: %v", err)
		http.Error(w, "Failed to fetch all schedules", http.StatusInternalServerError)
		return
	}
	allSchedules := make([]scheduleView, 0, len(allRows))
	for _, row := range allRows {
		allSchedules = append(allSchedules, vaccineToScheduleView(row))
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved all vaccine schedules", r.Method, r.RequestURI, http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schedules": allSchedules,
	})
}

// matchingCoop 1 แถวผลลัพธ์ของ GetVaccineMatchingCoopsHandler = คอกที่ต้องให้วัคซีนตัวนี้
// Status: "due" = อายุอยู่ในช่วงพอดี, "overdue" = อายุเลย max_age_days แล้วแต่ยังไม่เคยให้
type matchingCoop struct {
	CoopID         int    `json:"coop_id"`
	NameCoop       string `json:"name_coop"`
	CurrentAgeDays int    `json:"current_age_days"`
	Status         string `json:"status"`
}

// GetVaccineMatchingCoopsHandler (GET /api/vaccines/schedule/matching-coops?id=5)
// ตอบคำถาม "วัคซีนตัวนี้ต้องให้คอกไหนบ้าง"
//  1. โหลดประเภทวัคซีน (เฉพาะของผู้ใช้) ไม่เจอ -> 404
//  2. ดึงคอกที่เคยให้วัคซีนตัวนี้แล้วจาก vaccine_history ทำเป็น map
//  3. วนทุกคอก ข้ามคอกที่: ไม่มีวันเกิด/วันรับเข้า, ให้ไปแล้ว, ยังไม่ถึงวันรับเข้าเลี้ยง, อายุยังไม่ถึงขั้นต่ำ
//  4. ที่เหลือคือคอกที่ต้องให้ ระบุว่า due หรือ overdue
//
// คอกที่ไม่มีชื่อ แสดงเป็น "คอกที่ <id>"
func GetVaccineMatchingCoopsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}
	scheduleID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	var schedule models.Vaccine
	if err := database.DB.Where("vaccine_id = ? AND user_id = ?", scheduleID, userID).First(&schedule).Error; err != nil {
		log.Printf("[%s] %s - %d (Vaccine schedule not found: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
		http.Error(w, "Vaccine schedule not found", http.StatusNotFound)
		return
	}

	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve coops", http.StatusInternalServerError)
		return
	}

	type vaccineHistoryRow struct {
		CoopID int `gorm:"column:coop_id"`
	}
	var histories []vaccineHistoryRow
	database.DB.Model(&models.VaccineHistory{}).Select("coop_id").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?) AND name_vaccine = ?", userID, schedule.Name).
		Find(&histories)
	given := make(map[int]bool, len(histories))
	for _, h := range histories {
		given[h.CoopID] = true
	}

	now := time.Now()
	matches := []matchingCoop{}

	for _, coop := range coops {
		if coop.Birthday.IsZero() || coop.DateAdoptAnimals.IsZero() {
			continue
		}
		if given[coop.CoopID] {
			continue
		}
		// ยังไม่ถึงวันนำเข้าเลี้ยง - ข้ามไปก่อน (เหมือนเงื่อนไขใน GetRecommendedVaccinesHandler)
		if models.DateKey(now) < models.DateKey(coop.DateAdoptAnimals) {
			continue
		}
		ageInDays := models.CalculateChickenAge(coop.Birthday)
		if ageInDays < schedule.MinAgeDays {
			continue // ยังไม่ถึงอายุขั้นต่ำ
		}
		status := "due"
		if ageInDays > schedule.MaxAgeDays {
			status = "overdue"
		}
		name := coop.NameCoop
		if name == "" {
			name = "คอกที่ " + strconv.Itoa(coop.CoopID)
		}
		matches = append(matches, matchingCoop{
			CoopID:         coop.CoopID,
			NameCoop:       name,
			CurrentAgeDays: ageInDays,
			Status:         status,
		})
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ %d coops match vaccine schedule ID %d", r.Method, r.RequestURI, http.StatusOK, len(matches), scheduleID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vaccine_id":     schedule.VaccineID,
		"vaccine_name":   schedule.Name,
		"min_age_days":   schedule.MinAgeDays,
		"max_age_days":   schedule.MaxAgeDays,
		"matching_coops": matches,
	})
}

// CalendarAlertResponse 1 รายการแจ้งเตือนวัคซีนบนปฏิทิน
// ID เป็นรูปแบบ "<coop_id>_<ชื่อวัคซีน>" เช่น "26_vitaminB5" ใช้อ้างอิงตอนกดเสร็จสิ้น/ลบ
// Date = วันครบกำหนด (YYYY-MM-DD), IsCompleted = ให้แล้วหรือยัง
type CalendarAlertResponse struct {
	ID            string `json:"id"`
	Date          string `json:"date"`
	CoopID        string `json:"coop_id"`
	VaccineName   string `json:"vaccine_name"`
	InjectionType string `json:"injection_type"`
	IsCompleted   bool   `json:"is_completed"`
	IsOverdue     bool   `json:"is_overdue"`
	ChickenAge    int    `json:"chicken_age"`
	Description   string `json:"description"`
}

// GetVaccineCalendarAlertsHandler (/api/vaccines/alerts) ฟังก์ชันเดียวรองรับ 3 method:
//
// DELETE ?id=<coop>_<ชื่อ> : ลบ "ประเภท" วัคซีนชื่อนี้ (ของผู้ใช้) และประวัติการให้ชื่อนี้ทั้งหมด
//
//	วัคซีนตัวนี้จะไม่แจ้งเตือนอีก จำกัดด้วย user_id กันไปลบของฟาร์มอื่นที่ชื่อซ้ำ
//
// PUT ?id=<coop>_<ชื่อ> : กด "เสร็จสิ้น" / "ยกเลิก" (body: is_completed, method, chicken_age, note)
//   - เสร็จสิ้น: ถ้ายังไม่มีประวัติ จะเช็กว่าถึงวันครบกำหนดหรือยัง (ยังไม่ถึง -> 400 กดก่อนไม่ได้)
//     แล้วสร้างแถวใน vaccine_history (ใช้วันเกิดไก่จริงของคอก ไม่ใช่เวลาที่กด)
//   - ยกเลิก: ลบประวัติของ (คอก, วัคซีน) นั้น
//
// GET : สร้างรายการแจ้งเตือนทุกคอก x ทุกประเภทวัคซีน ตามหลักวันครบกำหนดด้านบนของไฟล์
//
//	และใส่ IsCompleted จากประวัติ (IsOverdue ส่ง false เสมอ ปฏิทินหลักใช้ /api/calendar/markers ที่คำนวณจริง)
//
// ข้อสังเกต: แยก id ด้วย "_" ถ้าชื่อวัคซีนมี "_" จะได้ชื่อแค่ส่วนแรก
func GetVaccineCalendarAlertsHandler(w http.ResponseWriter, r *http.Request) {

	// ==========================================
	// 🌟 1. จัดการคำสั่ง DELETE (ลบข้อมูลวัคซีนออกจากฐานข้อมูล)
	// ==========================================
	if r.Method == http.MethodDelete {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.UserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		alertID := r.URL.Query().Get("id")
		if alertID == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		// แยก id (เช่น "26_vitaminB5" -> parts[0]="26", parts[1]="vitaminB5")
		parts := strings.Split(alertID, "_")
		if len(parts) < 2 {
			http.Error(w, "Invalid id format", http.StatusBadRequest)
			return
		}

		vaccineName := parts[1] // ดึงชื่อวัคซีนออกมา

		// 🛑 ลบแถว "เกณฑ์/ประเภท" ในตาราง vaccine เอง เพื่อไม่ให้วัคซีนตัวนี้ไปแจ้งเตือน
		// อีก - จำกัดเฉพาะของ user ที่เรียกเท่านั้น กันไปลบประเภทวัคซีนชื่อเดียวกัน
		// ของฟาร์มอื่นโดยไม่ได้ตั้งใจ
		err := database.DB.Where("name_vaccine = ? AND user_id = ?", vaccineName, userID).Delete(&models.Vaccine{}).Error
		if err != nil {
			log.Printf("Failed to delete vaccine schedule: %v", err)
			http.Error(w, "Failed to delete schedule", http.StatusInternalServerError)
			return
		}

		// 🛑 ลบประวัติการให้วัคซีนที่เคยมีอยู่ในตาราง vaccine_history ด้วย - เฉพาะคอกของผู้เรียกเท่านั้น
		database.DB.Where("name_vaccine = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", vaccineName, userID).Delete(&models.VaccineHistory{})

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Deleted vaccine schedule: %s", r.Method, r.RequestURI, http.StatusOK, vaccineName)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "ลบข้อมูลวัคซีนสำเร็จ",
			"id":      alertID,
		})
		return
	}

	// ==========================================
	// 🌟 2. จัดการคำสั่ง PUT (เปลี่ยนสถานะ กดเสร็จสิ้น/ยกเลิก)
	// ==========================================
	if r.Method == http.MethodPut {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := auth.UserIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		alertID := r.URL.Query().Get("id")
		if alertID == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		parts := strings.Split(alertID, "_")
		if len(parts) < 2 {
			http.Error(w, "Invalid id format", http.StatusBadRequest)
			return
		}

		coopID, err := strconv.Atoi(parts[0])
		if err != nil {
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		coop, err := database.GetCoopByIDForUser(coopID, userID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				http.Error(w, "Coop not found", http.StatusNotFound)
				return
			}
			log.Printf("Failed to verify coop ownership: %v", err)
			http.Error(w, "Failed to verify coop ownership", http.StatusInternalServerError)
			return
		}
		vaccineName := parts[1]

		var body struct {
			IsCompleted bool   `json:"is_completed"`
			Method      string `json:"method"`
			ChickenAge  int    `json:"chicken_age"`
			Note        string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if body.IsCompleted {
			var count int64
			database.DB.Model(&models.VaccineHistory{}).Where("coop_id = ? AND name_vaccine = ?", coopID, vaccineName).Count(&count)
			if count == 0 {
				// 🛑 ห้ามกด "สำเร็จแล้ว" ก่อนถึงวันครบกำหนดจริง (เดิมไม่เช็คเลย กด
				// ได้ทุกเมื่อแม้ยังไม่ถึงวันที่ควรให้จริงๆ) - คำนวณวันครบกำหนดแบบ
				// เดียวกับ GET /api/vaccines/alerts (ดู GetVaccineCalendarAlertsHandler
				// ด้านล่าง) ย้อนกลับมาเช็คเฉยๆ ไม่กระทบแถวที่กดสำเร็จไปแล้วก่อนหน้า
				var schedule models.Vaccine
				if err := database.DB.Where("name_vaccine = ? AND user_id = ?", vaccineName, userID).First(&schedule).Error; err == nil {
					dueDate := coop.Birthday.AddDate(0, 0, schedule.MinAgeDays)
					if models.DateKey(dueDate) < models.DateKey(coop.DateAdoptAnimals) {
						dueDate = coop.DateAdoptAnimals
					}
					if models.DateKey(time.Now()) < models.DateKey(dueDate) {
						http.Error(w, "ยังไม่ถึงวันครบกำหนดให้วัคซีนนี้ กดสำเร็จก่อนไม่ได้", http.StatusBadRequest)
						return
					}
				}

				// Birthday ต้องเป็นวันเกิดไก่จริงของคอกนี้ (เหมือนที่ CreateVaccine
				// ในวาซีนrepository ทำ) ไม่ใช่เวลาปัจจุบันตอนกดให้วัคซีน - ของเดิมใส่
				// &now ผิด ทำให้ birthday ของประวัติวัคซีนคลาดเคลื่อนจากวันเกิดจริง
				now := time.Now()
				history := models.VaccineHistory{
					CoopID:         coopID,
					NameCoop:       coop.NameCoop,
					Birthday:       coop.Birthday,
					Name:           vaccineName,
					Method:         body.Method,
					RecommendedAge: strconv.Itoa(body.ChickenAge),
					Note:           body.Note,
					RecordDate:     now,
				}
				if err := database.DB.Create(&history).Error; err != nil {
					log.Printf("Failed to create vaccine history record: %v", err)
					http.Error(w, "Failed to save status", http.StatusInternalServerError)
					return
				}
			}
		} else {
			err := database.DB.Where("coop_id = ? AND name_vaccine = ?", coopID, vaccineName).Delete(&models.VaccineHistory{}).Error
			if err != nil {
				log.Printf("Failed to delete vaccine history record: %v", err)
				http.Error(w, "Failed to delete status", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Updated alert status ID: %s to %t", r.Method, r.RequestURI, http.StatusOK, alertID, body.IsCompleted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":      "อัปเดตสถานะสำเร็จ",
			"id":           alertID,
			"is_completed": body.IsCompleted,
		})
		return
	}

	// ==========================================
	// 🌟 3. จัดการคำสั่ง GET (ดึงข้อมูลปฏิทินไปแสดง)
	// ==========================================
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve coops", http.StatusInternalServerError)
		return
	}

	var schedules []models.Vaccine
	if err := database.DB.Where("user_id = ?", userID).Find(&schedules).Error; err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve schedules: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve schedules", http.StatusInternalServerError)
		return
	}

	type vaccineHistoryRow struct {
		CoopID      int    `gorm:"column:coop_id"`
		VaccineName string `gorm:"column:name_vaccine"`
	}
	var histories []vaccineHistoryRow
	database.DB.Model(&models.VaccineHistory{}).Select("coop_id, name_vaccine").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Find(&histories)

	completedMap := make(map[string]bool)
	for _, h := range histories {
		key := strconv.Itoa(h.CoopID) + "_" + h.VaccineName
		completedMap[key] = true
	}

	var alerts []CalendarAlertResponse

	for _, coop := range coops {
		if coop.Birthday.IsZero() || coop.DateAdoptAnimals.IsZero() {
			continue
		}

		for _, schedule := range schedules {
			windowStart := coop.Birthday.AddDate(0, 0, schedule.MinAgeDays)
			windowEnd := coop.Birthday.AddDate(0, 0, schedule.MaxAgeDays)

			// ถ้าวันสุดท้ายของช่วงอายุที่ควรให้วัคซีนนี้ (max_age_days) ผ่านไปแล้ว
			// ตั้งแต่ก่อนวันที่รับไก่เข้าคอก แปลว่าไก่โตเกินเงื่อนไขนี้มาก่อนจะมาถึงฟาร์มเรา
			// (น่าจะเคยได้รับวัคซีนตัวนี้จากที่อื่นมาแล้ว) จึงไม่ต้องขึ้นให้ฟาร์มนี้ต้องฉีดอีก
			if models.DateKey(windowEnd) < models.DateKey(coop.DateAdoptAnimals) {
				continue
			}

			// ถ้าช่วงอายุเริ่มต้น (min_age_days) มาก่อนวันนำเข้าเลี้ยง แต่ยังไม่เลย max_age_days
			// (รับไก่เข้ามาตอนอายุอยู่กลางช่วงพอดี) ให้ถือว่าครบกำหนดตั้งแต่วันที่รับเข้าเลี้ยง
			// เพราะฟาร์มเราเพิ่งมีไก่ตัวนี้ตอนนั้น ให้ฉีดได้ทันทีที่รับมา
			dueDate := windowStart
			if models.DateKey(dueDate) < models.DateKey(coop.DateAdoptAnimals) {
				dueDate = coop.DateAdoptAnimals
			}

			alertID := strconv.Itoa(coop.CoopID) + "_" + schedule.Name
			isDone := completedMap[alertID]

			alerts = append(alerts, CalendarAlertResponse{
				ID:            alertID,
				Date:          models.DateKey(dueDate),
				CoopID:        strconv.Itoa(coop.CoopID),
				VaccineName:   schedule.Name,
				InjectionType: schedule.Method,
				IsCompleted:   isDone,
				IsOverdue:     false,
				ChickenAge:    schedule.MinAgeDays,
				Description:   schedule.Note,
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved %d calendar alerts", r.Method, r.RequestURI, http.StatusOK, len(alerts))
	json.NewEncoder(w).Encode(alerts)
}

// NotificationResponse 1 รายการแจ้งเตือนวัคซีนที่ส่งให้แอป Flutter
// (ชื่อวัคซีน, ชื่อคอก, วันที่ dd/mm/yyyy, เป็นของวันนี้หรือพรุ่งนี้)
type NotificationResponse struct {
	Name       string `json:"name"`
	CoopName   string `json:"coop_name"`
	RecordDate string `json:"record_date"`
	IsToday    bool   `json:"is_today"` // เพิ่มตัวแปรนี้เพื่อบอกแอปว่าเป็นของวันนี้ใช่หรือไม่
}

// GetVaccineNotificationsHandler (GET /api/notifications/vaccines) หาวัคซีนที่ครบกำหนด "วันนี้" และ "พรุ่งนี้"
// วนทุกคอก x ทุกประเภทวัคซีน:
//   - ข้ามคอกที่พรุ่งนี้ยังไม่ถึงวันรับเข้าเลี้ยง และวัคซีนที่ช่วงอายุจบก่อนรับเข้าเลี้ยง
//   - อายุไก่วันนี้ == min_age_days -> แจ้งเตือนของวันนี้
//   - อายุไก่พรุ่งนี้ == min_age_days -> แจ้งเตือนของพรุ่งนี้
//
// ใช้แสดงในหน้าแจ้งเตือนของแอป ไม่มีเลยส่ง [] (ไม่ใช่ null กันแอปแครช)
func GetVaccineNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	notifications := []NotificationResponse{}

	coops, err := database.GetAllCoops(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve coops", http.StatusInternalServerError)
		return
	}

	var schedules []models.Vaccine
	if err := database.DB.Where("user_id = ?", userID).Find(&schedules).Error; err != nil {
		http.Error(w, "Failed to retrieve schedules", http.StatusInternalServerError)
		return
	}

	// หาวันที่ปัจจุบัน (วันนี้) และ วันพรุ่งนี้
	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)

	todayStr := today.Format("02/01/2006")
	tomorrowStr := tomorrow.Format("02/01/2006")

	for _, coop := range coops {
		// ต้องมีวันเกิด และวันนำเข้า
		if coop.Birthday.IsZero() || coop.DateAdoptAnimals.IsZero() {
			continue
		}

		// 🛑 ถ้าพรุ่งนี้ยังเป็นช่วงเวลาก่อนที่ไก่จะถูกนำเข้าเลี้ยง ก็ข้ามเล้านี้ไปเลย (เทียบ YYYY-MM-DD)
		if models.DateKey(tomorrow) < models.DateKey(coop.DateAdoptAnimals) {
			continue
		}

		// คำนวณอายุไก่สำหรับ "วันนี้" และ "วันพรุ่งนี้" (เทียบจากวันเกิด)
		ageInDaysToday := int(today.Sub(coop.Birthday).Hours() / 24)
		ageInDaysTomorrow := ageInDaysToday + 1

		for _, schedule := range schedules {
			// 🛑 กรอง: ถ้าช่วงอายุที่ควรให้วัคซีนนี้ (ถึง max_age_days) ผ่านไปหมดแล้ว
			// ตั้งแต่ก่อนวันนำเข้าไก่ แปลว่าไก่โตเกินเงื่อนไขนี้มาก่อนถึงฟาร์มเรา ไม่ต้องแจ้งเตือน
			windowEnd := coop.Birthday.AddDate(0, 0, schedule.MaxAgeDays)
			if models.DateKey(windowEnd) < models.DateKey(coop.DateAdoptAnimals) {
				continue
			}

			coopName := "คอก " + strconv.Itoa(coop.CoopID)

			// ตรวจสอบคิวของ "วันนี้"
			if ageInDaysToday == schedule.MinAgeDays {
				notifications = append(notifications, NotificationResponse{
					Name:       schedule.Name,
					CoopName:   coopName,
					RecordDate: todayStr,
					IsToday:    true,
				})
			}

			// ตรวจสอบคิวของ "วันพรุ่งนี้"
			if ageInDaysTomorrow == schedule.MinAgeDays {
				notifications = append(notifications, NotificationResponse{
					Name:       schedule.Name,
					CoopName:   coopName,
					RecordDate: tomorrowStr,
					IsToday:    false,
				})
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notifications)
}

// GetHealthCheckNotiHandler (GET /api/notifications/health_checks) แจ้งเตือนตรวจสุขภาพวันนี้/พรุ่งนี้
// ข้อมูลจาก database.GetHealthCheckNotifications ไม่มีเลยส่ง [] กันแอปแครช
func GetHealthCheckNotiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// เรียกฟังก์ชันดึงข้อมูลที่เราเพิ่งสร้าง
	notifications, err := database.GetHealthCheckNotifications(userID)
	if err != nil {
		http.Error(w, "Failed to fetch health check notifications", http.StatusInternalServerError)
		return
	}

	// ถ้าไม่มีคอกที่ต้องตรวจเลย ให้ตอบกลับเป็น Array ว่าง ป้องกันแอปแครช
	if notifications == nil {
		notifications = []database.HealthNotification{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notifications)

	log.Printf("✓ Retrieved %d health check notifications", len(notifications))
}

// UpdateCustomMedicineHandler (PUT /api/vaccines/schedule/update?old_name=ชื่อเดิม) แก้ไขประเภทวัคซีน/ยา
//  1. อ้างแถวด้วย "ชื่อเดิม" + user_id แล้วอัปเดตชื่อ วิธีให้ ช่วงอายุ คำอธิบาย
//  2. ถ้าเปลี่ยนชื่อ อัปเดตชื่อในประวัติการให้ (vaccine_history) ของคอกผู้ใช้ด้วย
//     ทำไม: ระบบจับคู่ประวัติกับประเภทด้วยชื่อ ถ้าไม่แก้ตาม วัคซีนที่ให้ไปแล้วจะกลายเป็น "ยังไม่ให้"
//
// จำกัดด้วย user_id ทั้งสองขั้น กันไปแก้ของฟาร์มอื่นที่ตั้งชื่อวัคซีนซ้ำกัน
func UpdateCustomMedicineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	oldName := r.URL.Query().Get("old_name")
	if oldName == "" {
		http.Error(w, "Missing old_name parameter", http.StatusBadRequest)
		return
	}

	// 🌟 สร้างโครงสร้างมารับค่าชั่วคราว เพื่อตรวจสอบชนิดข้อมูลจาก Flutter ได้แม่นยำขึ้น
	var body struct {
		Name        string `json:"name"`
		MinAgeDays  int    `json:"min_age_days"`
		MaxAgeDays  int    `json:"max_age_days"`
		Method      string `json:"method"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("Decode error: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log ดูค่าที่ส่งมาจาก Flutter เพื่อเช็คความชัวร์ใน Console ของ Go
	log.Printf("📥 รับค่าแก้ไข: %s -> Name: %s, MinAge: %d, MaxAge: %d", oldName, body.Name, body.MinAgeDays, body.MaxAgeDays)

	// 🌟 อัปเดตแถว "ประเภท" ในตาราง vaccine เอง - จำกัดเฉพาะของ user ที่เรียกเท่านั้น
	err := database.DB.Model(&models.Vaccine{}).Where("name_vaccine = ? AND user_id = ?", oldName, userID).Updates(map[string]interface{}{
		"name_vaccine": body.Name,
		"method":       body.Method,
		"min_age_days": body.MinAgeDays, // ส่งค่า int ตรงเข้าฐานข้อมูล
		"max_age_days": body.MaxAgeDays, // ส่งค่า int ตรงเข้าฐานข้อมูล
		"note":         body.Description,
	}).Error

	if err != nil {
		log.Printf("❌ Failed to update schedule: %v", err)
		http.Error(w, "Failed to update schedule", http.StatusInternalServerError)
		return
	}

	// 🌟 อัปเดตชื่อในตาราง vaccine_history (ประวัติให้จริงที่ผูกกับคอก) ด้วย เผื่อ
	// ผู้ใช้แก้ไขชื่อวัคซีน ไม่ให้ชื่อในประวัติเก่าค้างเป็นชื่อเดิม - จำกัดเฉพาะคอก
	// ของ user ที่เรียกเท่านั้น กันไปทับชื่อประวัติของฟาร์มอื่นที่บังเอิญตั้งชื่อ
	// วัคซีนซ้ำกัน
	if body.Name != oldName {
		database.DB.Model(&models.VaccineHistory{}).
			Where("name_vaccine = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", oldName, userID).
			Updates(map[string]interface{}{
				"name_vaccine": body.Name,
			})
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("✅ อัปเดตข้อมูลวัคซีนสำเร็จในระบบ")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "อัปเดตข้อมูลวัคซีนสำเร็จ",
	})
}
