// =====================================================================================
// ไฟล์ handlers/egg.go - API บันทึกการเก็บไข่ (เพิ่ม / ดู / แก้ไข / ลบ)
// =====================================================================================
//
// ข้อมูลอยู่ใน database/egg_repository.go
// GET /api/eggs แยกที่ handleEggsGet ใน routes.go:
//
//	?id= -> GetEggHandler, ?coop_id= -> GetEggsByCoopHandler, ไม่มี -> GetAllEggsHandler
//
// id ส่งผ่าน query string (?id=) ทุกตัว
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// CreateEggHandler (POST /api/eggs) บันทึกการเก็บไข่ (หน้า "บันทึกการเก็บไข่")
//   - ต้องมี coop_id, วันที่เก็บ, จำนวนไข่ (ไม่เป็น 0) ไม่งั้น 400
//   - คอกไม่ใช่ของผู้ใช้ -> 404
//   - ไก่อายุยังไม่ถึง 5 เดือน -> 422 พร้อมข้อความบอกวันที่เริ่มเก็บไข่ได้
//     (422 = ข้อมูลรูปแบบถูก แต่ผิดเงื่อนไขทางธุรกิจ)
//   - คอกนี้บันทึกไข่ของวันนั้นไปแล้ว -> 409
//   - สำเร็จ -> 201
func CreateEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateEggRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.CoopID == 0 || req.DateCollectEgg.IsZero() || req.NumberEgg == 0 {
		http.Error(w, "Missing required fields: coop_id, date_collect_egg, number_egg", http.StatusBadRequest)
		return
	}

	egg, err := database.CreateEgg(&req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		if tooYoung, ok := database.IsChickensTooYoung(err); ok {
			http.Error(w, tooYoung.Error(), http.StatusUnprocessableEntity)
			return
		}
		if database.IsDuplicateEggDate(err) {
			http.Error(w, "Egg record for this coop and date already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(egg)
}

// GetEggHandler (GET /api/eggs?id=) ดึงบันทึกไข่ 1 รายการ (เฉพาะของผู้ใช้) ไม่เจอ -> 404
func GetEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract ID from query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing egg id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid egg id", http.StatusBadRequest)
		return
	}

	egg, err := database.GetEggByID(id, userID)
	if err != nil {
		http.Error(w, "Egg record not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(egg)
}

// GetEggsByCoopHandler (GET /api/eggs?coop_id=) ดึงบันทึกไข่ทั้งหมดของคอกเดียว
// คอกไม่ใช่ของผู้ใช้ -> 404, ไม่มีข้อมูลส่ง [] แทน null
func GetEggsByCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract coop_id from query parameter
	coopIDStr := r.URL.Query().Get("coop_id")
	if coopIDStr == "" {
		http.Error(w, "Missing coop_id parameter", http.StatusBadRequest)
		return
	}

	coopID, err := strconv.Atoi(coopIDStr)
	if err != nil {
		http.Error(w, "Invalid coop_id", http.StatusBadRequest)
		return
	}

	eggs, err := database.GetEggsByCoopID(coopID, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if eggs == nil {
		eggs = []models.Egg{}
	}
	json.NewEncoder(w).Encode(eggs)
}

// GetAllEggsHandler (GET /api/eggs) ดึงบันทึกไข่ของทุกคอก ใช้ทำกราฟ/สรุปจำนวนไข่
func GetAllEggsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	eggs, err := database.GetAllEggs(userID)
	if err != nil {
		http.Error(w, "Failed to fetch egg records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if eggs == nil {
		eggs = []models.Egg{}
	}
	json.NewEncoder(w).Encode(eggs)
}

// UpdateEggHandler (PUT /api/eggs?id=) แก้ไขบันทึกไข่
// ไม่เจอ -> 404, เปลี่ยนเป็นวัน/คอกที่ไก่ยังอายุไม่ถึง -> 422
func UpdateEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
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
		http.Error(w, "Missing egg id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid egg id", http.StatusBadRequest)
		return
	}

	var req models.UpdateEggRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	egg, err := database.UpdateEgg(id, &req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Egg record not found", http.StatusNotFound)
			return
		}
		if tooYoung, ok := database.IsChickensTooYoung(err); ok {
			http.Error(w, tooYoung.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(egg)
}

// DeleteEggHandler (DELETE /api/eggs?id=) ลบบันทึกไข่ 1 รายการ ไม่เจอ -> 404
func DeleteEggHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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
		http.Error(w, "Missing egg id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid egg id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteEgg(id, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Egg record not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete egg record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Egg record deleted successfully"})
}
