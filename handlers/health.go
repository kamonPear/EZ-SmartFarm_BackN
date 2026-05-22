package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Server is running", r.Method, r.RequestURI, http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "Server is running"}`))
}

// Handlers for Health CRUD are implemented in database package access

// CreateHealthHandler creates a new health record
// POST /api/healths
func CreateHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CoopID == 0 || req.RecordDate.IsZero() {
		http.Error(w, "Missing required fields: coop_id, record_date, healthy, poor_health", http.StatusBadRequest)
		return
	}

	health, err := database.CreateHealth(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(health)
}

// GetHealthHandler retrieves a single health record by ID
// GET /api/healths?id={id}
func GetHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing health id parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid health id", http.StatusBadRequest)
		return
	}
	h, err := database.GetHealthByID(id)
	if err != nil {
		http.Error(w, "Health record not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h)
}

// GetHealthsByCoopHandler retrieves health records for a coop
// GET /api/healths?coop_id={coop_id}
func GetHealthsByCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	coopIDStr := r.URL.Query().Get("coop_id")
	if coopIDStr == "" {
		http.Error(w, "Missing coop_id parameter", http.StatusBadRequest)
		return
	}
	coopID, err := strconv.Atoi(coopIDStr)
	if err != nil {
		http.Error(w, "Invalid coop_id", http.StatusBadRequest)
		return
	}
	hs, err := database.GetHealthsByCoopID(coopID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if hs == nil {
		hs = []models.Health{}
	}
	json.NewEncoder(w).Encode(hs)
}

// GetAllHealthsHandler retrieves all health records
// GET /api/healths
func GetAllHealthsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hs, err := database.GetAllHealths()
	if err != nil {
		http.Error(w, "Failed to fetch health records", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if hs == nil {
		hs = []models.Health{}
	}
	json.NewEncoder(w).Encode(hs)
}

// UpdateHealthHandler updates an existing health record
// PUT /api/healths?id={id}
func UpdateHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing health id parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid health id", http.StatusBadRequest)
		return
	}
	var req models.UpdateHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	h, err := database.UpdateHealth(id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h)
}

// DeleteHealthHandler deletes a health record
// DELETE /api/healths?id={id}
func DeleteHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing health id parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid health id", http.StatusBadRequest)
		return
	}
	if err := database.DeleteHealth(id); err != nil {
		http.Error(w, "Failed to delete health record", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Health record deleted successfully"})
}
