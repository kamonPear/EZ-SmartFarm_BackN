package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateCoopHandler creates a new coop
// POST /api/coops
func CreateCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateCoopRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DateAdoptAnimals.IsZero() || req.Amount == 0 || req.Birthday.IsZero() {
		http.Error(w, "Missing required fields: date_adopt_animals, amount, birthday", http.StatusBadRequest)
		return
	}

	coop, err := database.CreateCoop(&req)
	if err != nil {
		http.Error(w, "Failed to create coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(coop)
}

// GetCoopHandler retrieves a single coop by ID
// GET /api/coops/{id}
func GetCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	coop, err := database.GetCoopByID(id)
	if err != nil {
		http.Error(w, "Coop not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coop)
}

// GetAllCoopsHandler retrieves all coops
// GET /api/coops
func GetAllCoopsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	coops, err := database.GetAllCoops()
	if err != nil {
		http.Error(w, "Failed to fetch coops", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coops)
}

// UpdateCoopHandler updates an existing coop
// PUT /api/coops/{id}
func UpdateCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	var req models.UpdateCoopRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	coop, err := database.UpdateCoop(id, &req)
	if err != nil {
		http.Error(w, "Failed to update coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coop)
}

// DeleteCoopHandler deletes a coop
// DELETE /api/coops/{id}
func DeleteCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteCoop(id); err != nil {
		http.Error(w, "Failed to delete coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Coop deleted successfully"})
}
