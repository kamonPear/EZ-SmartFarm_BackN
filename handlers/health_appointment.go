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

// CreateHealthAppointmentHandler books a manual health-check appointment for a coop
// POST /api/health-appointments  body: {"coop_id":1,"appointment_date":"2026-10-05T00:00:00Z"}
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

// GetAllHealthAppointmentsHandler lists every upcoming manual appointment for the user
// GET /api/health-appointments
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

// GetHealthAppointmentsByCoopHandler lists upcoming manual appointments for one coop
// GET /api/health-appointments?coop_id=5
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

// DeleteHealthAppointmentHandler removes a manual appointment
// DELETE /api/health-appointments?id=3
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
