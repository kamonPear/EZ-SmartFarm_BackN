package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
	"EZ-SmartFarm_BachN/services"
)

// GetFarmThresholdHandler retrieves the caller's farm-wide target temperature/ammonia
// GET /api/farm-threshold
func GetFarmThresholdHandler(w http.ResponseWriter, r *http.Request) {
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

	threshold, err := database.GetFarmThreshold(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch farm threshold: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch farm threshold", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(threshold)
}

// UpdateFarmThresholdHandler sets the caller's farm-wide target temperature/ammonia
// PUT /api/farm-threshold  body: {"temperature": 25, "ammonia": 35}
func UpdateFarmThresholdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.UpdateFarmThresholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	threshold, err := database.UpdateFarmThreshold(&req, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to update farm threshold: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update farm threshold", http.StatusInternalServerError)
		return
	}

	// ส่งค่าใหม่ไปให้ Arduino ทุกคอกของผู้ใช้ (retained) ทำใน goroutine ไม่ให้หน้าเว็บรอ broker
	go services.PublishThresholdsForUser(userID)

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Farm threshold set to %.1f°C / %.1f ppm", r.Method, r.RequestURI, http.StatusOK, req.Temperature, req.Ammonia)
	json.NewEncoder(w).Encode(threshold)
}
