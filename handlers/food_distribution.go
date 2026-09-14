package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// RecordFoodDistributionHandler is the single action that both deducts foodstock and
// records how it was split across coops - there is no fixed daily amount anymore,
// the deducted total is exactly the sum of every item's kg_given the caller provides.
// Stock is deducted first; the distribution rows are only written once that succeeds,
// so a failed deduction never leaves an orphaned distribution record behind.
// POST /api/foods/distribution
func RecordFoodDistributionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RecordFoodDistributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !models.IsValidFoodType(req.FoodType) {
		log.Printf("[%s] %s - %d (Invalid food_type: %q)", r.Method, r.RequestURI, http.StatusBadRequest, req.FoodType)
		http.Error(w, "ต้องระบุ food_type เป็น 'เม็ดเล็ก' หรือ 'เม็ดใหญ่'", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		log.Printf("[%s] %s - %d (No items provided)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "กรุณากรอกจำนวนกิโลของอย่างน้อย 1 คอก", http.StatusBadRequest)
		return
	}

	var total float64
	for _, item := range req.Items {
		if item.CoopID == 0 || item.KgGiven < 0 {
			log.Printf("[%s] %s - %d (Invalid item: coop_id=%d kg_given=%.2f)", r.Method, r.RequestURI, http.StatusBadRequest, item.CoopID, item.KgGiven)
			http.Error(w, "ข้อมูลคอก/จำนวนกิโลไม่ถูกต้อง", http.StatusBadRequest)
			return
		}
		total += item.KgGiven
	}

	if total <= 0 {
		log.Printf("[%s] %s - %d (Total is zero)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "ยอดรวมต้องมากกว่า 0", http.StatusBadRequest)
		return
	}

	if err := database.DeductFoodstockByType(req.FoodType, total); err != nil {
		log.Printf("[%s] %s - %d (Failed to deduct stock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to deduct stock", http.StatusInternalServerError)
		return
	}

	rows, err := database.CreateFoodDistributionBatch(req.FoodType, req.Items)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to save distribution: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to save distribution", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	log.Printf("[%s] %s - %d ✓ Deducted %.2f kg of %s and recorded distribution for %d coops", r.Method, r.RequestURI, http.StatusCreated, total, req.FoodType, len(rows))
	json.NewEncoder(w).Encode(rows)
}

// GetFoodDistributionHistoryHandler lists every recorded distribution, newest first.
// GET /api/foods/distribution?food_type=เม็ดเล็ก (food_type optional)
func GetFoodDistributionHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	foodType := r.URL.Query().Get("food_type")
	rows, err := database.GetFoodDistributionHistory(foodType)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch distribution history: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch distribution history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if rows == nil {
		rows = []models.FoodDistribution{}
	}
	log.Printf("[%s] %s - %d ✓ Retrieved %d food distribution records", r.Method, r.RequestURI, http.StatusOK, len(rows))
	json.NewEncoder(w).Encode(rows)
}
