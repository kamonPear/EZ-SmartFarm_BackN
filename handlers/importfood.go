package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateImportFoodHandler records a new food import lot and adds it onto the current foodstock total
// POST /api/importfoods
func CreateImportFoodHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateImportFoodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ImportVolume < 0 || req.ExpiryDate.IsZero() {
		log.Printf("[%s] %s - %d (Missing required fields)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing or invalid required fields: import_volume, expiry_date", http.StatusBadRequest)
		return
	}

	lot, err := database.CreateImportFood(&req)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to create import food lot: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to create import food lot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	log.Printf("[%s] %s - %d ✓ Created import food lot ID: %d", r.Method, r.RequestURI, http.StatusCreated, lot.LotID)
	json.NewEncoder(w).Encode(lot)
}

// GetImportFoodHandler retrieves a single import food lot by ID
// GET /api/importfoods?id=1
func GetImportFoodHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid import food id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid import food id", http.StatusBadRequest)
		return
	}

	lot, err := database.GetImportFoodByID(id)
	if err != nil {
		log.Printf("[%s] %s - %d (Import food lot not found)", r.Method, r.RequestURI, http.StatusNotFound)
		http.Error(w, "Import food lot not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lot)
}

// GetAllImportFoodsHandler retrieves the full import food history
// GET /api/importfoods
func GetAllImportFoodsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lots, err := database.GetAllImportFood()
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch import food history: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch import food history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d import food records", r.Method, r.RequestURI, http.StatusOK, len(lots))
	json.NewEncoder(w).Encode(lots)
}
