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

// DayMarkerDetail is one line item within a day - structured (not a pre-joined
// string) so each client can render its own icon/color/tap-target per item instead
// of just dumping prose. Category/Status use the same small fixed vocabularies on
// both clients:
//
//	category: "vaccine" | "health" | "birthday" | "adopt" | "appointment"
//	status:   "done" | "upcoming" | "overdue" | "info"
type DayMarkerDetail struct {
	Text     string `json:"text"`
	Category string `json:"category"`
	Status   string `json:"status"`
	CoopID   int    `json:"coop_id"`
}

// DayMarker is what each date key in the /api/calendar/markers response maps to.
// Field names are snake_case (backend convention) - the Angular/Flutter clients map
// them onto their own camelCase DayMarker types, same shape the web app used to
// compute itself client-side from 3 separate requests.
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

// GetCalendarMarkersHandler is the single source of truth for "what's happening on
// each day" across the whole app - vaccine due/overdue/completed, health checks,
// coop birthdays/adoption dates, and manual health appointments, all merged into one
// map keyed by "YYYY-MM-DD". Both the web app and the Flutter app call this instead
// of each re-deriving the same thing from 3-4 separate endpoints themselves.
// GET /api/calendar/markers?coop_id=5 (coop_id optional - omit for the whole farm)
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
	var schedules []models.MedicineSchedule
	if err := database.DB.Find(&schedules).Error; err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve schedules: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve schedules", http.StatusInternalServerError)
		return
	}

	type vaccineHistory struct {
		CoopID      int    `gorm:"column:coop_id"`
		VaccineName string `gorm:"column:name_vaccine"`
	}
	var histories []vaccineHistory
	database.DB.Model(&models.Vaccine{}).Select("coop_id, name_vaccine").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Find(&histories)
	completedMap := make(map[string]bool, len(histories))
	for _, h := range histories {
		completedMap[strconv.Itoa(h.CoopID)+"_"+h.VaccineName] = true
	}

	todayKey := time.Now().Format("2006-01-02")

	for _, coop := range relevantCoops {
		if coop.Birthday.IsZero() || coop.DateAdoptAnimals.IsZero() {
			continue
		}
		for _, schedule := range schedules {
			windowEnd := coop.Birthday.AddDate(0, 0, schedule.MaxAgeDays)
			if windowEnd.Format("2006-01-02") < coop.DateAdoptAnimals.Format("2006-01-02") {
				continue
			}
			dueDate := coop.Birthday.AddDate(0, 0, schedule.MinAgeDays)
			if dueDate.Format("2006-01-02") < coop.DateAdoptAnimals.Format("2006-01-02") {
				dueDate = coop.DateAdoptAnimals
			}
			dueKey := dueDate.Format("2006-01-02")
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
		m := getOrCreate(h.RecordDate.Format("2006-01-02"))
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
			m := getOrCreate(coop.Birthday.Format("2006-01-02"))
			m.HasCoopInfo = true
			m.Details = append(m.Details, DayMarkerDetail{
				Text:     fmt.Sprintf("🎂 วันเกิดไก่ – %s", coopName(coop.CoopID)),
				Category: "birthday",
				Status:   "info",
				CoopID:   coop.CoopID,
			})
		}
		if !coop.DateAdoptAnimals.IsZero() {
			m := getOrCreate(coop.DateAdoptAnimals.Format("2006-01-02"))
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
		m := getOrCreate(a.AppointmentDate.Format("2006-01-02"))
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
