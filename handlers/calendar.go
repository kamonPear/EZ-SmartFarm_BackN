// =====================================================================================
// ไฟล์ handlers/calendar.go - API จุดแสดงเหตุการณ์บนปฏิทิน (หน้าปฏิทินของทั้งแอปและเว็บ)
// =====================================================================================
//
// ปฏิทินแสดงเหตุการณ์ 5 แบบในแต่ละวัน:
//
//	วัคซีน (ถึงกำหนด/ให้แล้ว/เกินกำหนด), ตรวจสุขภาพ, วันเกิดไก่, วันรับเข้าเลี้ยง, นัดตรวจที่ตั้งเอง
//
// เดิมแอปและเว็บต่างดึง 3-4 API แล้วคำนวณเอง ทำให้ผลไม่ตรงกัน
// จึงย้ายมาคำนวณที่ backend ที่เดียว ทั้งสองฝั่งได้ผลเหมือนกันแน่นอน
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// DayMarkerDetail เหตุการณ์ 1 รายการในวันนั้น
//   - Text     : ข้อความที่แสดง เช่น "วัคซีนนิวคาสเซิล – คอก A (ถึงกำหนด)"
//   - Category : vaccine | health | birthday | adopt | appointment (ใช้เลือกไอคอน/สี)
//   - Status   : done | upcoming | overdue | info
//   - CoopID   : คอกที่เกี่ยวข้อง (กดแล้วไปหน้าคอกนั้นได้)
//
// ส่งเป็นโครงสร้าง ไม่ใช่ข้อความรวมก้อนเดียว เพื่อให้แต่ละฝั่งแสดงไอคอน/สีเองได้
type DayMarkerDetail struct {
	Text     string `json:"text"`
	Category string `json:"category"`
	Status   string `json:"status"`
	CoopID   int    `json:"coop_id"`
}

// DayMarker สรุปของ 1 วัน (key เป็น "YYYY-MM-DD")
//   - VaccineStatus : "due" = มีวัคซีนที่ยังไม่ให้, "done" = ให้ครบแล้ว, ไม่มี field = ไม่มีวัคซีนวันนั้น
//   - HasHealth / HasCoopInfo / HasHealthAppointment : มีเหตุการณ์ประเภทนั้นไหม (ใช้วาดจุดสี)
//   - Details : รายการเหตุการณ์ทั้งหมดของวันนั้น
type DayMarker struct {
	// "due" if any vaccine for that date isn't completed yet (or is overdue);
	// "done" if every vaccine alert on that date has been completed. Omitted
	// entirely if no vaccine alert falls on that date.
	VaccineStatus        string            `json:"vaccine_status,omitempty"`
	HasHealth            bool              `json:"has_health,omitempty"`
	HasCoopInfo          bool              `json:"has_coop_info,omitempty"`
	HasHealthAppointment bool              `json:"has_health_appointment,omitempty"`
	Details              []DayMarkerDetail `json:"details"`
}

// GetCalendarMarkersHandler (GET /api/calendar/markers?coop_id=5) สร้างข้อมูลปฏิทินทั้งหมด
// ไม่ใส่ coop_id = ทั้งฟาร์ม ผลลัพธ์เป็น map วันที่ -> DayMarker
//
// ขั้นตอน:
//  1. ดึงคอกทั้งหมดของผู้ใช้ (ถ้ากรองคอก เหลือคอกเดียว)
//  2. วัคซีน: วนทุกคอก x ทุกประเภทวัคซีน
//     - วันกำหนด = วันเกิดไก่ + อายุขั้นต่ำที่ต้องให้ (MinAgeDays)
//     - ถ้าช่วงให้วัคซีนจบก่อนวันรับเข้าเลี้ยง = ฟาร์มอื่นให้ไปแล้ว ข้าม
//     - ถ้าวันกำหนดอยู่ก่อนวันรับเข้าเลี้ยง เลื่อนเป็นวันรับเข้าเลี้ยง
//     - เช็กประวัติวัคซีนว่าให้แล้วหรือยัง (completedMap key = "coopID_ชื่อวัคซีน")
//     ยังไม่ให้และเลยวันแล้ว = เกินกำหนด (overdue)
//     ใช้สูตรเดียวกับ GetVaccineCalendarAlertsHandler ใน vaccine.go
//  3. ผลตรวจสุขภาพ: ใส่จุดในวันที่ตรวจ
//  4. วันเกิดไก่ และวันรับเข้าเลี้ยงของแต่ละคอก
//  5. นัดตรวจสุขภาพที่ผู้ใช้ตั้งเอง
//
// getOrCreate = ถ้าวันนั้นยังไม่มีใน map ก็สร้างใหม่ ทำให้รวมเหตุการณ์หลายแบบในวันเดียวได้
// เปรียบเทียบวันด้วย models.DateKey (ข้อความ YYYY-MM-DD) เพื่อไม่ให้เวลา/เขตเวลามีผล
func GetCalendarMarkersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var filterCoopID *int
	if v := r.URL.Query().Get("coop_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid coop_id", http.StatusBadRequest)
			return
		}
		filterCoopID = &id
	}

	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve coops", http.StatusInternalServerError)
		return
	}

	coopName := func(id int) string {
		for _, c := range coops {
			if c.CoopID == id {
				if c.NameCoop != "" {
					return c.NameCoop
				}
				break
			}
		}
		return fmt.Sprintf("คอกที่ %d", id)
	}

	relevantCoops := coops
	if filterCoopID != nil {
		relevantCoops = nil
		for _, c := range coops {
			if c.CoopID == *filterCoopID {
				relevantCoops = append(relevantCoops, c)
			}
		}
	}

	markers := make(map[string]*DayMarker)
	getOrCreate := func(key string) *DayMarker {
		if m, exists := markers[key]; exists {
			return m
		}
		m := &DayMarker{Details: []DayMarkerDetail{}}
		markers[key] = m
		return m
	}

	// ----- วัคซีน (คำนวณแบบเดียวกับ GetVaccineCalendarAlertsHandler ทุกประการ -
	// ต่างแค่ตรงนี้คำนวณ is_overdue จริงๆ แทนที่จะ hardcode false) -----
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
	completedMap := make(map[string]bool, len(histories))
	for _, h := range histories {
		completedMap[strconv.Itoa(h.CoopID)+"_"+h.VaccineName] = true
	}

	todayKey := models.DateKey(time.Now())

	for _, coop := range relevantCoops {
		if coop.Birthday.IsZero() || coop.DateAdoptAnimals.IsZero() {
			continue
		}
		for _, schedule := range schedules {
			windowEnd := coop.Birthday.AddDate(0, 0, schedule.MaxAgeDays)
			if models.DateKey(windowEnd) < models.DateKey(coop.DateAdoptAnimals) {
				continue
			}
			dueDate := coop.Birthday.AddDate(0, 0, schedule.MinAgeDays)
			if models.DateKey(dueDate) < models.DateKey(coop.DateAdoptAnimals) {
				dueDate = coop.DateAdoptAnimals
			}
			dueKey := models.DateKey(dueDate)
			isDone := completedMap[strconv.Itoa(coop.CoopID)+"_"+schedule.Name]
			isOverdue := !isDone && dueKey < todayKey

			m := getOrCreate(dueKey)
			if !isDone {
				m.VaccineStatus = "due"
			} else if m.VaccineStatus != "due" {
				m.VaccineStatus = "done"
			}

			status := "upcoming"
			label := "ถึงกำหนด"
			if isDone {
				status = "done"
				label = "ให้แล้ว"
			} else if isOverdue {
				status = "overdue"
				label = "เกินกำหนด"
			}
			m.Details = append(m.Details, DayMarkerDetail{
				Text:     fmt.Sprintf("วัคซีน%s – %s (%s)", schedule.Name, coopName(coop.CoopID), label),
				Category: "vaccine",
				Status:   status,
				CoopID:   coop.CoopID,
			})
		}
	}

	// ----- ประวัติสุขภาพ -----
	healthQuery := database.DB.Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID)
	if filterCoopID != nil {
		healthQuery = healthQuery.Where("coop_id = ?", *filterCoopID)
	}
	var healths []models.Health
	if err := healthQuery.Find(&healths).Error; err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve health records: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve health records", http.StatusInternalServerError)
		return
	}
	for _, h := range healths {
		m := getOrCreate(models.DateKey(h.RecordDate))
		m.HasHealth = true
		m.Details = append(m.Details, DayMarkerDetail{
			Text:     fmt.Sprintf("ตรวจสุขภาพ – %s", coopName(h.CoopID)),
			Category: "health",
			Status:   "info",
			CoopID:   h.CoopID,
		})
	}

	// ----- วันเกิดไก่ / วันที่รับเข้าเลี้ยงของคอก -----
	for _, coop := range relevantCoops {
		if !coop.Birthday.IsZero() {
			m := getOrCreate(models.DateKey(coop.Birthday))
			m.HasCoopInfo = true
			m.Details = append(m.Details, DayMarkerDetail{
				Text:     fmt.Sprintf("🎂 วันเกิดไก่ – %s", coopName(coop.CoopID)),
				Category: "birthday",
				Status:   "info",
				CoopID:   coop.CoopID,
			})
		}
		if !coop.DateAdoptAnimals.IsZero() {
			m := getOrCreate(models.DateKey(coop.DateAdoptAnimals))
			m.HasCoopInfo = true
			m.Details = append(m.Details, DayMarkerDetail{
				Text:     fmt.Sprintf("🏠 วันที่รับเข้าเลี้ยง – %s", coopName(coop.CoopID)),
				Category: "adopt",
				Status:   "info",
				CoopID:   coop.CoopID,
			})
		}
	}

	// ----- นัดตรวจสุขภาพที่กำหนดเอง -----
	apptQuery := database.DB.Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID)
	if filterCoopID != nil {
		apptQuery = apptQuery.Where("coop_id = ?", *filterCoopID)
	}
	var appts []models.HealthAppointment
	if err := apptQuery.Find(&appts).Error; err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve health appointments: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve health appointments", http.StatusInternalServerError)
		return
	}
	for _, a := range appts {
		m := getOrCreate(models.DateKey(a.AppointmentDate))
		m.HasHealthAppointment = true
		m.Details = append(m.Details, DayMarkerDetail{
			Text:     fmt.Sprintf("🗓️ นัดตรวจสุขภาพ (กำหนดเอง) – %s", coopName(a.CoopID)),
			Category: "appointment",
			Status:   "upcoming",
			CoopID:   a.CoopID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Built calendar markers for %d days", r.Method, r.RequestURI, http.StatusOK, len(markers))
	json.NewEncoder(w).Encode(markers)
}
