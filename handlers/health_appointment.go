// =====================================================================================
// ไฟล์ handlers/health_appointment.go - API นัดตรวจสุขภาพไก่ที่ผู้ใช้นัดเอง
// =====================================================================================
//
// ผู้ใช้กดเลือกวันในปฏิทินเพื่อนัดตรวจสุขภาพคอก (ตาราง health_appointments)
// นัดเหล่านี้แสดงเป็นจุดบนปฏิทิน (handlers/calendar.go) และในรายการแจ้งเตือน
// ข้อมูลอยู่ใน database/health_appointment_repository.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateHealthAppointmentHandler (POST /api/health-appointments) สร้างนัดตรวจสุขภาพ
// body: {"coop_id":1, "appointment_date":"2026-10-05T00:00:00Z"}
// ต้องมีทั้ง coop_id และวันที่ ไม่งั้น 400 / คอกไม่ใช่ของผู้ใช้ -> 404 / สำเร็จ -> 201
func CreateHealthAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateHealthAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.CoopID <= 0 || req.AppointmentDate.IsZero() {
		log.Printf("[%s] %s - %d (Missing coop_id/appointment_date)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "coop_id and appointment_date are required", http.StatusBadRequest)
		return
	}

	appt, err := database.CreateHealthAppointment(&req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to create health appointment: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to create health appointment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Health appointment created for coop %d on %s", r.Method, r.RequestURI, http.StatusCreated, appt.CoopID, appt.AppointmentDate.Format("2006-01-02"))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(appt)
}

// GetAllHealthAppointmentsHandler (GET /api/health-appointments) ดึงนัดที่ยังมาไม่ถึงของทุกคอก
// ถ้าไม่มีเลย ส่ง [] (ไม่ใช่ null) หน้าเว็บ/แอปจะวนลูปได้โดยไม่ error
// ถูกเรียกผ่าน: handleHealthAppointmentsGet ใน routes.go (เมื่อไม่มี ?coop_id=)
func GetAllHealthAppointmentsHandler(w http.ResponseWriter, r *http.Request) {
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

	appts, err := database.GetHealthAppointments(userID, nil)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve health appointments: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve health appointments", http.StatusInternalServerError)
		return
	}
	if appts == nil {
		appts = []models.HealthAppointment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appts)
}

// GetHealthAppointmentsByCoopHandler (GET /api/health-appointments?coop_id=5) ดึงนัดของคอกเดียว
// ส่ง &coopID (pointer) ให้ database เพื่อบอกว่า "กรองตามคอก" (ถ้าส่ง nil = ทุกคอก)
// ถูกเรียกผ่าน: handleHealthAppointmentsGet ใน routes.go (เมื่อมี ?coop_id=)
func GetHealthAppointmentsByCoopHandler(w http.ResponseWriter, r *http.Request) {
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

	coopID, err := strconv.Atoi(r.URL.Query().Get("coop_id"))
	if err != nil {
		http.Error(w, "invalid coop_id", http.StatusBadRequest)
		return
	}

	appts, err := database.GetHealthAppointments(userID, &coopID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve health appointments: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve health appointments", http.StatusInternalServerError)
		return
	}
	if appts == nil {
		appts = []models.HealthAppointment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appts)
}

// DeleteHealthAppointmentHandler (DELETE /api/health-appointments?id=3) ยกเลิกนัด
// ไม่เจอ หรือไม่ใช่ของผู้ใช้ -> 404
func DeleteHealthAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteHealthAppointment(id, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Appointment not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Appointment not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to delete health appointment: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to delete health appointment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Health appointment %d deleted", r.Method, r.RequestURI, http.StatusOK, id)
	json.NewEncoder(w).Encode(map[string]string{"message": "ลบนัดตรวจสุขภาพสำเร็จ"})
}
