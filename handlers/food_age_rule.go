package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetFoodAgeRulesHandler returns the caller's feeding table (empty list if none yet)
// GET /api/food-age-rules
func GetFoodAgeRulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rules, err := database.GetFoodAgeRules(userID)
	if err != nil {
		http.Error(w, "Failed to fetch food age rules", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// UpdateFoodAgeRulesHandler replaces the caller's whole feeding table
// PUT /api/food-age-rules
// body: {"rules":[{"food_type":"เม็ดเล็ก","min_weeks":0,"max_weeks":14,"grams_per_bird":60}, ...]}
func UpdateFoodAgeRulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req models.UpdateFoodAgeRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if msg := req.Validate(); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	rules, err := database.ReplaceFoodAgeRules(userID, req.Rules)
	if err != nil {
		http.Error(w, "Failed to save food age rules", http.StatusInternalServerError)
		return
	}
	log.Printf("[%s] %s - %d ✓ Saved %d food age rules for user %d", r.Method, r.RequestURI, http.StatusOK, len(rules), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}
