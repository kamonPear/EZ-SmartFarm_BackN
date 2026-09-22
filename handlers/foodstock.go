package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetFoodstockHandler retrieves a single foodstock by ID
// GET /api/foodstocks?id=1
func GetFoodstockHandler(w http.ResponseWriter, r *http.Request) {
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

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing foodstock id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing foodstock id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid foodstock id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid foodstock id", http.StatusBadRequest)
		return
	}

	foodstock, err := database.GetFoodstockByID(id, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Foodstock not found)", r.Method, r.RequestURI, http.StatusNotFound)
		http.Error(w, "Foodstock not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved foodstock ID: %d", r.Method, r.RequestURI, http.StatusOK, foodstock.FoodID)
	json.NewEncoder(w).Encode(foodstock)
}

// GetAllFoodstocksHandler retrieves all foodstocks owned by the caller
// GET /api/foodstocks
func GetAllFoodstocksHandler(w http.ResponseWriter, r *http.Request) {
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

	foodstocks, err := database.GetAllFoodstocks(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch foodstocks: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch foodstocks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d foodstocks", r.Method, r.RequestURI, http.StatusOK, len(foodstocks))
	json.NewEncoder(w).Encode(foodstocks)
}

// UpdateFoodstockHandler manually corrects the current stock total
// PUT /api/foodstocks?id=1
func UpdateFoodstockHandler(w http.ResponseWriter, r *http.Request) {
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

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing foodstock id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing foodstock id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid foodstock id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid foodstock id", http.StatusBadRequest)
		return
	}

	var req models.UpdateFoodstockRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	foodstock, err := database.UpdateFoodstock(id, &req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Foodstock not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Foodstock not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to update foodstock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update foodstock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Updated foodstock ID: %d", r.Method, r.RequestURI, http.StatusOK, foodstock.FoodID)
	json.NewEncoder(w).Encode(foodstock)
}

// DeleteFoodstockHandler deletes a foodstock
// DELETE /api/foodstocks?id=1
func DeleteFoodstockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing foodstock id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing foodstock id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid foodstock id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid foodstock id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteFoodstock(id, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Foodstock not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Foodstock not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to delete foodstock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to delete foodstock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Deleted foodstock ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
	json.NewEncoder(w).Encode(map[string]string{"message": "Foodstock deleted successfully"})
}

// GetFoodHistoryHandler retrieves the caller's history of food imports (importfood table)
// GET /api/food_history
func GetFoodHistoryHandler(w http.ResponseWriter, r *http.Request) {
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

	history, err := database.GetAllImportFood(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch food history: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch food history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d food history records", r.Method, r.RequestURI, http.StatusOK, len(history))
	json.NewEncoder(w).Encode(history)
}

// DailyDeductAmounts คือจำนวน กก./วัน ที่ตัดออกจากสต็อกแต่ละประเภท (ค่าเดียวกับที่ Cron Job ใช้)
var DailyDeductAmounts = map[string]float64{
	models.FoodTypeSmallPellet: 20.0,
	models.FoodTypeLargePellet: 30.0,
}

// ForceDeductStockHandler อนุญาตให้เรียก API เพื่อตัดสต็อกแบบแมนนวล ต้องระบุประเภทอาหารก่อนตัดเสมอ
// ตัดเฉพาะสต็อกของผู้เรียก (userID) เท่านั้น
// POST /api/foodstocks/force-deduct  body: {"food_type": "เม็ดเล็ก"}
func ForceDeductStockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.DeductFoodstockRequest
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

	amount := DailyDeductAmounts[req.FoodType]

	if err := database.DeductFoodstockByType(req.FoodType, userID, amount); err != nil {
		log.Printf("[%s] %s - %d (Failed to deduct stock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to deduct stock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ ตัดสต็อก %s %.0f กก. เรียบร้อยแล้ว", r.Method, r.RequestURI, http.StatusOK, req.FoodType, amount)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   fmt.Sprintf("หักสต็อกอาหาร%s %.0f กิโลกรัม สำเร็จ", req.FoodType, amount),
		"food_type": req.FoodType,
		"deducted":  amount,
	})
}
