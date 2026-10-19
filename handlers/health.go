// =====================================================================================
// ไฟล์ handlers/health.go - API ผลตรวจสุขภาพไก่ และ API เช็กว่าเซิร์ฟเวอร์ทำงานอยู่
// =====================================================================================
//
// ผลตรวจ 1 รายการ = คอก, วันที่ตรวจ, จำนวนไก่สุขภาพดี, จำนวนไก่ป่วย, หมายเหตุ
// ข้อมูลอยู่ใน database/health_repository.go
// GET /api/healths แยกไปฟังก์ชันต่าง ๆ ตาม query string ที่ handleHealthsGet ใน routes.go:
//
//	?id=       -> GetHealthHandler
//	?coop_id=  -> GetHealthsByCoopHandler
//	ไม่มีเลย   -> GetAllHealthsHandler
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

// HealthCheck (/health) ตอบ {"status":"ok"} เสมอ ใช้เช็กว่าเซิร์ฟเวอร์ยังทำงานอยู่
// ไม่เกี่ยวกับ "สุขภาพไก่" Render (ที่ deploy) และบริการปลุกเซิร์ฟเวอร์ใช้ยิงเช็กเป็นระยะ
// ไม่ต้องล็อกอิน และไม่แตะฐานข้อมูล จึงตอบได้เร็ว
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Server is running", r.Method, r.RequestURI, http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "Server is running"}`))
}

// Handlers for Health CRUD are implemented in database package access

// CreateHealthHandler (POST /api/healths) บันทึกผลตรวจสุขภาพใหม่
//   - ต้องมี coop_id และวันที่ตรวจ ไม่งั้น 400
//   - คอกไม่ใช่ของผู้ใช้ -> 404
//   - คอกนี้มีผลตรวจของวันนั้นแล้ว (IsDuplicateHealthDate) -> 409 ให้ไปแก้ไขรายการเดิมแทน
//   - สำเร็จ -> 201
func CreateHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CoopID == 0 || req.RecordDate.IsZero() {
		http.Error(w, "Missing required fields: coop_id, record_date, healthy, poor_health", http.StatusBadRequest)
		return
	}

	health, err := database.CreateHealth(&req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		if database.IsDuplicateHealthDate(err) {
			http.Error(w, "Health record for this coop and date already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(health)
}

// GetHealthHandler (GET /api/healths?id=) ดึงผลตรวจ 1 รายการ (เฉพาะของผู้ใช้) ไม่เจอ -> 404
func GetHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
		http.Error(w, "Missing health id parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid health id", http.StatusBadRequest)
		return
	}
	h, err := database.GetHealthByID(id, userID)
	if err != nil {
		http.Error(w, "Health record not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h)
}

// GetHealthsByCoopHandler (GET /api/healths?coop_id=) ดึงผลตรวจทั้งหมดของคอกเดียว
// ใช้ในส่วนประวัติสุขภาพของหน้าแก้ไขข้อมูลคอก ไม่มีข้อมูลส่ง [] แทน null
func GetHealthsByCoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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
	hs, err := database.GetHealthsByCoopID(coopID, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if hs == nil {
		hs = []models.Health{}
	}
	json.NewEncoder(w).Encode(hs)
}

// GetAllHealthsHandler (GET /api/healths) ดึงผลตรวจของทุกคอกของผู้ใช้ ใช้ทำกราฟสรุปสุขภาพ
func GetAllHealthsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	hs, err := database.GetAllHealths(userID)
	if err != nil {
		http.Error(w, "Failed to fetch health records", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if hs == nil {
		hs = []models.Health{}
	}
	json.NewEncoder(w).Encode(hs)
}

// UpdateHealthHandler (PUT /api/healths?id=) แก้ไขผลตรวจ (แก้เฉพาะช่องที่ส่งมา)
// ไม่เจอหรือไม่ใช่ของผู้ใช้ -> 404
func UpdateHealthHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Missing health id parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid health id", http.StatusBadRequest)
		return
	}
	var req models.UpdateHealthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	h, err := database.UpdateHealth(id, &req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Health record not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h)
}

// DeleteHealthHandler (DELETE /api/healths?id=) ลบผลตรวจ 1 รายการ ไม่เจอ -> 404
func DeleteHealthHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Missing health id parameter", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid health id", http.StatusBadRequest)
		return
	}
	if err := database.DeleteHealth(id, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Health record not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete health record", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Health record deleted successfully"})
}
