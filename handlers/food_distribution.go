// =====================================================================================
// ไฟล์ handlers/food_distribution.go - API บันทึกการแจกอาหารให้แต่ละคอก (ตัดสต็อกด้วยมือ)
// =====================================================================================
//
// ผู้ใช้กรอกว่าวันนี้ให้อาหารแต่ละคอกไปกี่ กก. ระบบจะตัดสต็อกตามยอดรวม
// และเก็บประวัติรายคอกไว้ในตาราง food_distribution
// ข้อมูลอยู่ใน database/food_distribution_repository.go และ foodstock_repository.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// RecordFoodDistributionHandler (POST /api/foods/distribution) ตัดสต็อกและบันทึกการแจกอาหาร
// body: {"food_type":"เม็ดเล็ก", "items":[{"coop_id":1, "kg_given":2.5}, ...]}
//
// ขั้นตอน:
//  1. ประเภทอาหารต้องถูกต้อง และต้องมีอย่างน้อย 1 คอก
//  2. วนทุกคอก: ค่าต้องถูกต้อง (kg ไม่ติดลบ), รวมยอด, และคอกต้องเป็นของผู้ใช้ (ไม่ใช่ -> 404)
//  3. ยอดรวมต้องมากกว่า 0
//  4. เช็กสต็อกก่อน ถ้าไม่พอ -> 400 พร้อมบอกว่ามีเท่าไร ขาดเท่าไร
//     ทำไม: ยอดนี้ผู้ใช้กรอกเองว่าให้ไปจริง ถ้ารับไว้ทั้งที่สต็อกไม่พอ ตัวเลขจะไม่ตรงความจริง
//     ผู้ใช้ควรรู้ทันทีว่าต้องเติมสต็อกก่อน
//  5. ตัดสต็อกก่อน แล้วค่อยบันทึกประวัติรายคอก
//     ทำไมลำดับนี้: ถ้าตัดสต็อกล้ม จะไม่มีประวัติที่ไม่มีการตัดจริงค้างอยู่
//
// ตอบ 201 พร้อมแถวประวัติที่บันทึก
func RecordFoodDistributionHandler(w http.ResponseWriter, r *http.Request) {
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

		owned, err := database.CoopBelongsToUser(item.CoopID, userID)
		if err != nil {
			log.Printf("[%s] %s - %d (Failed to verify coop ownership: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
			http.Error(w, "Failed to verify coop ownership", http.StatusInternalServerError)
			return
		}
		if !owned {
			log.Printf("[%s] %s - %d (Coop not found: %d)", r.Method, r.RequestURI, http.StatusNotFound, item.CoopID)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
	}

	if total <= 0 {
		log.Printf("[%s] %s - %d (Total is zero)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "ยอดรวมต้องมากกว่า 0", http.StatusBadRequest)
		return
	}

	// เช็คก่อนว่าสต็อกจริงมีพอไหม แล้วปฏิเสธถ้าไม่พอ แทนที่จะปล่อยให้ตัดแค่เท่าที่มี
	// เงียบๆ - ตรงนี้เป็นยอดที่ผู้ใช้ "กรอกเองตรงๆ" ว่าให้อาหารคอกไหนไปเท่าไรจริง
	// ถ้าระบบรับไว้ทั้งที่สต็อกไม่พอจะได้ตัวเลขที่ไม่ตรงความจริง (อ้างว่าให้ไปแล้ว
	// มากกว่าที่มีอยู่จริง) ผู้ใช้ควรรู้ทันทีว่าต้องเติมสต็อกก่อน ไม่ใช่มาเจอทีหลัง
	currentQty, found, err := database.GetFoodstockQuantity(userID, req.FoodType)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to check stock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to check stock", http.StatusInternalServerError)
		return
	}
	if !found || currentQty < total {
		log.Printf("[%s] %s - %d (Insufficient stock: have %.2f, need %.2f)", r.Method, r.RequestURI, http.StatusBadRequest, currentQty, total)
		http.Error(w, fmt.Sprintf("สต็อกอาหาร%sเหลือไม่พอ มีอยู่ %.2f กก. แต่กรอกรวม %.2f กก. กรุณาเติมสต็อกก่อน", req.FoodType, currentQty, total), http.StatusBadRequest)
		return
	}

	if _, _, err := database.DeductFoodstockByType(req.FoodType, userID, total); err != nil {
		log.Printf("[%s] %s - %d (Failed to deduct stock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to deduct stock", http.StatusInternalServerError)
		return
	}

	rows, err := database.CreateFoodDistributionBatch(req.FoodType, req.Items, userID)
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

// DeleteAllFoodDistributionHandler (DELETE /api/foods/distribution) ล้างประวัติการแจกอาหารทั้งหมดของผู้ใช้
// ลบแค่ประวัติ ไม่คืนยอดกลับเข้าสต็อก
func DeleteAllFoodDistributionHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := database.DeleteAllFoodDistribution(userID); err != nil {
		log.Printf("[%s] %s - %d (Failed to delete distribution history: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to delete distribution history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Cleared food distribution history", r.Method, r.RequestURI, http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Food distribution history cleared"})
}

// GetFoodDistributionHistoryHandler (GET /api/foods/distribution?food_type=เม็ดเล็ก)
// ดึงประวัติการแจกอาหาร เรียงจากล่าสุด กรองตามประเภทได้ (ไม่ใส่ = ทุกประเภท)
// ไม่มีข้อมูลส่ง [] แทน null
func GetFoodDistributionHistoryHandler(w http.ResponseWriter, r *http.Request) {
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

	foodType := r.URL.Query().Get("food_type")
	rows, err := database.GetFoodDistributionHistory(foodType, userID)
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
