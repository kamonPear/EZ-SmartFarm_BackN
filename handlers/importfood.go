// =====================================================================================
// ไฟล์ handlers/importfood.go - API นำเข้าอาหาร (ส่วน "นำเข้าอาหาร" ในหน้าคลังอาหาร)
// =====================================================================================
//
// นำเข้า 1 ครั้ง = 1 ล็อต (ตาราง importfood): ประเภท, ปริมาณ (กก.), วันหมดอายุ, ราคา ฯลฯ
// บันทึกล็อตแล้ว ยอดในคลัง (foodstock) ของประเภทนั้นจะเพิ่มขึ้นอัตโนมัติ
// ข้อมูลอยู่ใน database/importfood_repository.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateImportFoodHandler (POST /api/importfoods) บันทึกล็อตอาหารที่นำเข้าใหม่
// ตรวจข้อมูลก่อน:
//   - ปริมาณต้องไม่ติดลบ และต้องมีวันหมดอายุ
//   - ประเภทต้องเป็น "เม็ดเล็ก" หรือ "เม็ดใหญ่" (models.IsValidFoodType)
//     เพราะสต็อกแยกตามประเภทนี้ ถ้าพิมพ์ผิดจะไปสร้างสต็อกประเภทแปลก ๆ
//
// จากนั้น database.CreateImportFood บันทึกล็อตและบวกยอดเข้าคลังใน Transaction เดียว
// ตอบ 201 พร้อมข้อมูลล็อต
func CreateImportFoodHandler(w http.ResponseWriter, r *http.Request) {
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

	if !models.IsValidFoodType(req.FoodType) {
		log.Printf("[%s] %s - %d (Invalid food_type: %q)", r.Method, r.RequestURI, http.StatusBadRequest, req.FoodType)
		http.Error(w, "food_type must be either 'เม็ดเล็ก' or 'เม็ดใหญ่'", http.StatusBadRequest)
		return
	}

	lot, err := database.CreateImportFood(&req, userID)
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

// GetImportFoodHandler (GET /api/importfoods?id=1) ดึงล็อตเดียวตาม id (เฉพาะของผู้ใช้)
// ไม่เจอหรือเป็นของคนอื่น -> 404
// ถูกเรียกผ่าน: handleImportFoodsGet ใน routes.go (เมื่อมี ?id=)
func GetImportFoodHandler(w http.ResponseWriter, r *http.Request) {
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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid import food id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid import food id", http.StatusBadRequest)
		return
	}

	lot, err := database.GetImportFoodByID(id, userID)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Failed to fetch import food lot: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		}
		log.Printf("[%s] %s - %d (Import food lot not found)", r.Method, r.RequestURI, http.StatusNotFound)
		http.Error(w, "Import food lot not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lot)
}

// DeleteAllImportFoodsHandler (DELETE /api/importfoods) ล้างประวัติการนำเข้าทั้งหมดของผู้ใช้
// ลบแค่ "ประวัติ" ยอดคงเหลือในคลังไม่เปลี่ยน
func DeleteAllImportFoodsHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := database.DeleteAllImportFood(userID); err != nil {
		log.Printf("[%s] %s - %d (Failed to delete import food history: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to delete import food history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Cleared import food history", r.Method, r.RequestURI, http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Import food history cleared"})
}

// GetAllImportFoodsHandler (GET /api/importfoods) ดึงประวัติการนำเข้าทั้งหมดของผู้ใช้
// ใช้แสดงตาราง "ประวัติการนำเข้าอาหาร"
// ถูกเรียกผ่าน: handleImportFoodsGet ใน routes.go (เมื่อไม่มี ?id=)
func GetAllImportFoodsHandler(w http.ResponseWriter, r *http.Request) {
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

	lots, err := database.GetAllImportFood(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch import food history: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch import food history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d import food records", r.Method, r.RequestURI, http.StatusOK, len(lots))
	json.NewEncoder(w).Encode(lots)
}
