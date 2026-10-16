package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetFoodAgeConfigHandler returns the caller's food-type age ranges (default if unset)
// GET /api/food-age-config
func GetFoodAgeConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cfg, err := database.GetFoodAgeConfig(userID)
	if err != nil {
		http.Error(w, "Failed to fetch food age config", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// UpdateFoodAgeConfigHandler saves the caller's food-type age ranges
// PUT /api/food-age-config
// body: {"small_min_weeks":0,"small_max_weeks":14,"large_min_weeks":14,"large_max_weeks":0}
func UpdateFoodAgeConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req models.UpdateFoodAgeConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if msg := req.Validate(); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	cfg, err := database.UpdateFoodAgeConfig(&req, userID)
	if err != nil {
		http.Error(w, "Failed to save food age config", http.StatusInternalServerError)
		return
	}
	log.Printf("[%s] %s - %d ✓ Food age config saved for user %d", r.Method, r.RequestURI, http.StatusOK, userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}
