package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateVaccineHandler records a new vaccine administration for a coop
// POST /api/vaccines
func CreateVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

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

	coop, err := database.GetCoopByID(req.CoopID)
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

// GetVaccineHandler retrieves vaccine records (ประวัติการฉีดจริง)
// GET /api/vaccines
func GetVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

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

		vaccine, err := database.GetVaccineByID(id)
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

		vaccines, err := database.GetVaccinesByCoopID(id)
		if err != nil {
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
	vaccines, err := database.GetAllVaccines()
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve vaccines: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve vaccines", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved %d vaccines", r.Method, r.RequestURI, http.StatusOK, len(vaccines))
	json.NewEncoder(w).Encode(vaccines)
}

// DeleteVaccineHandler deletes a vaccine record
// DELETE /api/vaccines
func DeleteVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

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

		err = database.DeleteVaccine(id)
		if err != nil {
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

		err = database.DeleteVaccinesByCoopID(id)
		if err != nil {
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

// AddCustomMedicineHandler รับข้อมูล POST จากแอป Flutter มาบันทึกลงฐานข้อมูล
// POST /api/vaccines/schedule
func AddCustomMedicineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req models.MedicineSchedule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.DB.Create(&req).Error; err != nil {
		log.Printf("Failed to save medicine: %v", err)
		http.Error(w, "Failed to save medicine", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "บันทึกข้อมูลยา/วัคซีนสำเร็จ",
		"data":    req,
	})
}

// GetRecommendedVaccinesHandler ค้นหายาจาก Database ที่เหมาะสมกับอายุไก่
// GET /api/vaccines/recommended
func GetRecommendedVaccinesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	coopID := r.URL.Query().Get("coop_id")

	if coopID != "" {
		id, err := strconv.Atoi(coopID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid coop ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		coop, err := database.GetCoopByID(id)
		if err != nil {
			log.Printf("[%s] %s - %d (Coop not found: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}

		ageInDays := models.CalculateChickenAge(coop.Birthday)
		log.Printf("DEBUG: Coop ID %d - Birthday: %v, Current Age: %d days", id, coop.Birthday, ageInDays)

		var recommendedMedicines []models.MedicineSchedule

		// 🌟 ตรวจสอบว่าถึงวันที่นำไก่เข้าหรือยัง (เทียบแบบ YYYY-MM-DD ตัดปัญหาเรื่องเวลา)
		now := time.Now()
		if !coop.DateAdoptAnimals.IsZero() && now.Format("2006-01-02") < coop.DateAdoptAnimals.Format("2006-01-02") {
			// ถ้ายังไม่ถึงวันนำไก่เข้า ให้ส่งอาเรย์ว่างกลับไปเลย เพราะยังไม่ต้องแนะนำ
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"coop_id":              id,
				"birthday":             coop.Birthday,
				"current_age_days":     ageInDays,
				"recommended_vaccines": []models.MedicineSchedule{},
			})
			return
		}

		err = database.DB.Where("min_age_days <= ? AND max_age_days >= ?", ageInDays, ageInDays).Find(&recommendedMedicines).Error
		if err != nil {
			log.Printf("Failed to fetch medicines: %v", err)
			http.Error(w, "Failed to fetch medicines", http.StatusInternalServerError)
			return
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

	var allSchedules []models.MedicineSchedule
	if err := database.DB.Find(&allSchedules).Error; err != nil {
		log.Printf("Failed to fetch all schedules: %v", err)
		http.Error(w, "Failed to fetch all schedules", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved all vaccine schedules", r.Method, r.RequestURI, http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schedules": allSchedules,
	})
}



// โครงสร้างสำหรับส่งให้หน้าปฏิทิน Flutter
type CalendarAlertResponse struct {
	ID            string `json:"id"`
	Date          string `json:"date"`
	CoopID        string `json:"coop_id"`
	VaccineName   string `json:"vaccine_name"`
	InjectionType string `json:"injection_type"`
	IsCompleted   bool   `json:"is_completed"`
	ChickenAge    int    `json:"chicken_age"`
	Description   string `json:"description"`
}

// GetVaccineCalendarAlertsHandler ค้นหาจุดแจ้งเตือนของปฏิทินจาก Database
// รองรับ GET (ดึงข้อมูล), PUT (อัปเดตสถานะ), และ DELETE (ลบวัคซีน)
func GetVaccineCalendarAlertsHandler(w http.ResponseWriter, r *http.Request) {
	
	// ==========================================
	// 🌟 1. จัดการคำสั่ง DELETE (ลบข้อมูลวัคซีนออกจากฐานข้อมูล)
	// ==========================================
	if r.Method == http.MethodDelete {
		w.Header().Set("Content-Type", "application/json")

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

		// 🛑 ลบตารางเกณฑ์ (medicine_schedules) เพื่อไม่ให้วัคซีนตัวนี้ไปแจ้งเตือนอีก
		err := database.DB.Where("name = ?", vaccineName).Delete(&models.MedicineSchedule{}).Error
		if err != nil {
			log.Printf("Failed to delete vaccine schedule: %v", err)
			http.Error(w, "Failed to delete schedule", http.StatusInternalServerError)
			return
		}

		// 🛑 ลบประวัติการให้วัคซีนที่เคยมีอยู่ในตารางจริง (vaccine) ด้วย
		database.DB.Where("name_vaccine = ?", vaccineName).Delete(&models.Vaccine{})

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
			database.DB.Model(&models.Vaccine{}).Where("coop_id = ? AND name_vaccine = ?", coopID, vaccineName).Count(&count)
			if count == 0 {
				now := time.Now()
				vaccine := models.Vaccine{
					CoopID:         coopID,
					Name:           vaccineName,
					Method:         body.Method,
					RecommendedAge: strconv.Itoa(body.ChickenAge),
					Note:           body.Note,
					RecordDate:     now,
					Birthday:       &now,
				}
				if err := database.DB.Create(&vaccine).Error; err != nil {
					log.Printf("Failed to create vaccine record: %v", err)
					http.Error(w, "Failed to save status", http.StatusInternalServerError)
					return
				}
			}
		} else {
			err := database.DB.Where("coop_id = ? AND name_vaccine = ?", coopID, vaccineName).Delete(&models.Vaccine{}).Error
			if err != nil {
				log.Printf("Failed to delete vaccine record: %v", err)
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

	coops, err := database.GetAllCoops()
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve coops", http.StatusInternalServerError)
		return
	}

	var schedules []models.MedicineSchedule
	if err := database.DB.Find(&schedules).Error; err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve schedules: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve schedules", http.StatusInternalServerError)
		return
	}

	type VaccineHistory struct {
		CoopID      int    `gorm:"column:coop_id"`
		VaccineName string `gorm:"column:name_vaccine"`
	}
	var histories []VaccineHistory
	database.DB.Model(&models.Vaccine{}).Select("coop_id, name_vaccine").Find(&histories)

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
			dueDate := coop.Birthday.AddDate(0, 0, schedule.MinAgeDays)
			alertID := strconv.Itoa(coop.CoopID) + "_" + schedule.Name

			if dueDate.Format("2006-01-02") < coop.DateAdoptAnimals.Format("2006-01-02") {
				continue 
			}

			isDone := completedMap[alertID]

			alerts = append(alerts, CalendarAlertResponse{
				ID:            alertID,
				Date:          dueDate.Format("2006-01-02"),
				CoopID:        strconv.Itoa(coop.CoopID),
				VaccineName:   schedule.Name,
				InjectionType: schedule.Method,
				IsCompleted:   isDone,
				ChickenAge:    schedule.MinAgeDays,
				Description:   schedule.Description,
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved %d calendar alerts", r.Method, r.RequestURI, http.StatusOK, len(alerts))
	json.NewEncoder(w).Encode(alerts)
}

// โครงสร้างสำหรับตอบกลับการแจ้งเตือนให้แอป Flutter
type NotificationResponse struct {
	Name       string `json:"name"`
	CoopName   string `json:"coop_name"`
	RecordDate string `json:"record_date"`
	IsToday    bool   `json:"is_today"` // เพิ่มตัวแปรนี้เพื่อบอกแอปว่าเป็นของวันนี้ใช่หรือไม่
}

// GetVaccineNotificationsHandler คำนวณหาคิววัคซีนของ "วันนี้" และ "วันพรุ่งนี้"
// GET /api/notifications/vaccines
func GetVaccineNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	notifications := []NotificationResponse{}

	coops, err := database.GetAllCoops()
	if err != nil {
		http.Error(w, "Failed to retrieve coops", http.StatusInternalServerError)
		return
	}

	var schedules []models.MedicineSchedule
	if err := database.DB.Find(&schedules).Error; err != nil {
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
		if tomorrow.Format("2006-01-02") < coop.DateAdoptAnimals.Format("2006-01-02") {
			continue
		}

		// คำนวณอายุไก่สำหรับ "วันนี้" และ "วันพรุ่งนี้" (เทียบจากวันเกิด)
		ageInDaysToday := int(today.Sub(coop.Birthday).Hours() / 24)
		ageInDaysTomorrow := ageInDaysToday + 1

		for _, schedule := range schedules {
			// 🌟 คำนวณวันที่ต้องให้วัคซีน
			dueDate := coop.Birthday.AddDate(0, 0, schedule.MinAgeDays)

			// 🛑 กรอง: ข้ามวัคซีนที่กำหนดให้ ก่อนวันนำเข้าไก่ (เทียบ YYYY-MM-DD)
			if dueDate.Format("2006-01-02") < coop.DateAdoptAnimals.Format("2006-01-02") {
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

func GetHealthCheckNotiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// เรียกฟังก์ชันดึงข้อมูลที่เราเพิ่งสร้าง
	notifications, err := database.GetHealthCheckNotifications()
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

// UpdateCustomMedicineHandler อัปเดตข้อมูลเกณฑ์วัคซีน
// PUT /api/vaccines/schedule/update
// UpdateCustomMedicineHandler อัปเดตข้อมูลเกณฑ์วัคซีน
// PUT /api/vaccines/schedule/update
func UpdateCustomMedicineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
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

	// 🌟 บังคับอัปเดตตาราง medicine_schedules โดยชี้ฟิลด์แบบชัดเจน เจาะจงตารางด้วย .Table()
	err := database.DB.Table("medicine_schedules").Where("name = ?", oldName).Updates(map[string]interface{}{
		"name":         body.Name,
		"method":       body.Method,
		"min_age_days": body.MinAgeDays, // ส่งค่า int ตรงเข้าฐานข้อมูล
		"max_age_days": body.MaxAgeDays, // ส่งค่า int ตรงเข้าฐานข้อมูล
		"description":  body.Description,
	}).Error

	if err != nil {
		log.Printf("❌ Failed to update schedule: %v", err)
		http.Error(w, "Failed to update schedule", http.StatusInternalServerError)
		return
	}

	// 🌟 อัปเดตชื่อในตารางประวัติจริง (vaccine) ด้วย เผื่อผู้ใช้แก้ไขชื่อวัคซีน
	if body.Name != oldName {
		database.DB.Model(&models.Vaccine{}).Where("name_vaccine = ?", oldName).Updates(map[string]interface{}{
			"name_vaccine": body.Name,
			"name":         body.Name,
		})
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("✅ อัปเดตข้อมูลวัคซีนสำเร็จในระบบ")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "อัปเดตข้อมูลวัคซีนสำเร็จ",
	})
}