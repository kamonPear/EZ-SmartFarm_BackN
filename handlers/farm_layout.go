package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetFarmLayoutHandler retrieves the caller's chosen outline shape
// GET /api/farm-layout
func GetFarmLayoutHandler(w http.ResponseWriter, r *http.Request) {
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

	layout, err := database.GetFarmLayout(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch farm layout: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch farm layout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(layout)
}

// UpdateFarmLayoutHandler sets the caller's chosen outline shape
// PUT /api/farm-layout  body: {"shape": "circle"}
func UpdateFarmLayoutHandler(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateFarmLayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !models.IsValidFarmShape(req.Shape) {
		log.Printf("[%s] %s - %d (Invalid shape: %q)", r.Method, r.RequestURI, http.StatusBadRequest, req.Shape)
		http.Error(w, "shape must be 'circle', 'triangle', or 'square'", http.StatusBadRequest)
		return
	}

	layout, err := database.UpdateFarmLayoutShape(req.Shape, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to update farm layout: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update farm layout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Farm layout shape set to %s", r.Method, r.RequestURI, http.StatusOK, req.Shape)
	json.NewEncoder(w).Encode(layout)
}

// UpdateCoopPositionsHandler batch-saves every coop's position on the farm layout canvas
// PUT /api/coops/positions  body: {"positions": [{"coop_id":1,"pos_x":42.5,"pos_y":10.2}, ...]}
func UpdateCoopPositionsHandler(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateCoopPositionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.UpdateCoopPositions(req.Positions, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to update coop positions: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update coop positions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Saved positions for %d coops", r.Method, r.RequestURI, http.StatusOK, len(req.Positions))
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "บันทึกผังฟาร์มสำเร็จ", "count": len(req.Positions)})
}
