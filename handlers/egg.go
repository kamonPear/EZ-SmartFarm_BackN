package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateEggHandler creates a new egg record
// POST /api/eggs
func CreateEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateEggRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.CoopID == 0 || req.DateCollectEgg.IsZero() || req.NumberEgg == 0 {
		http.Error(w, "Missing required fields: coop_id, date_collect_egg, number_egg", http.StatusBadRequest)
		return
	}

	egg, err := database.CreateEgg(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(egg)
}

// GetEggHandler retrieves a single egg record by ID
// GET /api/eggs/{id}
func GetEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing egg id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid egg id", http.StatusBadRequest)
		return
	}

	egg, err := database.GetEggByID(id)
	if err != nil {
		http.Error(w, "Egg record not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(egg)
}

// GetEggsByCoopHandler retrieves all egg records for a specific coop
// GET /api/eggs/coop/{coop_id}
func GetEggsByCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract coop_id from query parameter
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

	eggs, err := database.GetEggsByCoopID(coopID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if eggs == nil {
		eggs = []models.Egg{}
	}
	json.NewEncoder(w).Encode(eggs)
}

// GetAllEggsHandler retrieves all egg records
// GET /api/eggs
func GetAllEggsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eggs, err := database.GetAllEggs()
	if err != nil {
		http.Error(w, "Failed to fetch egg records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if eggs == nil {
		eggs = []models.Egg{}
	}
	json.NewEncoder(w).Encode(eggs)
}

// UpdateEggHandler updates an existing egg record
// PUT /api/eggs/{id}
func UpdateEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing egg id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid egg id", http.StatusBadRequest)
		return
	}

	var req models.UpdateEggRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	egg, err := database.UpdateEgg(id, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(egg)
}

// DeleteEggHandler deletes an egg record
// DELETE /api/eggs/{id}
func DeleteEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing egg id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid egg id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteEgg(id); err != nil {
		http.Error(w, "Failed to delete egg record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Egg record deleted successfully"})
}
