package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateCoopHandler creates a new coop
// POST /api/coops
func CreateCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateCoopRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DateAdoptAnimals.IsZero() || req.Amount == 0 || req.Birthday.IsZero() {
		log.Printf("[%s] %s - %d (Missing required fields)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing required fields: date_adopt_animals, amount, birthday", http.StatusBadRequest)
		return
	}

	coop, err := database.CreateCoop(&req)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to create coop: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to create coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	log.Printf("[%s] %s - %d ✓ Created coop ID: %d", r.Method, r.RequestURI, http.StatusCreated, coop.CoopID)
	json.NewEncoder(w).Encode(coop)
}

// GetCoopHandler retrieves a single coop by ID
// GET /api/coops?id=1
func GetCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing coop id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid coop id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	coop, err := database.GetCoopByID(id)
	if err != nil {
		log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
		http.Error(w, "Coop not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved coop ID: %d", r.Method, r.RequestURI, http.StatusOK, coop.CoopID)
	json.NewEncoder(w).Encode(coop)
}

// GetAllCoopsHandler retrieves all coops
// GET /api/coops
func GetAllCoopsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	coops, err := database.GetAllCoops()
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch coops", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d coops", r.Method, r.RequestURI, http.StatusOK, len(coops))
	json.NewEncoder(w).Encode(coops)
}

// UpdateCoopHandler updates an existing coop
// PUT /api/coops?id=1
func UpdateCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing coop id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid coop id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	var req models.UpdateCoopRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	coop, err := database.UpdateCoop(id, &req)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to update coop: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Updated coop ID: %d", r.Method, r.RequestURI, http.StatusOK, coop.CoopID)
	json.NewEncoder(w).Encode(coop)
}

// DeleteCoopHandler deletes a coop
// DELETE /api/coops?id=1
func DeleteCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing coop id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid coop id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteCoop(id); err != nil {
		log.Printf("[%s] %s - %d (Failed to delete coop: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to delete coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Deleted coop ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
	json.NewEncoder(w).Encode(map[string]string{"message": "Coop deleted successfully"})
}
