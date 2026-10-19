// =====================================================================================
// ไฟล์ handlers/coop.go - API คอกไก่ (เพิ่ม / ดู / แก้ไข / ลบ)
// =====================================================================================
//
// ข้อมูลอยู่ใน database/coop_repository.go
// GET /api/coops แยกที่ handleCoopsGet ใน routes.go: มี ?id= -> GetCoopHandler, ไม่มี -> GetAllCoopsHandler
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

// CreateCoopHandler (POST /api/coops) เพิ่มคอกใหม่ (หน้า "เพิ่มคอกไก่")
//  1. ต้องกรอก ชื่อคอก วันรับเข้าเลี้ยง จำนวนไก่ วันเกิดไก่ ครบ ไม่งั้น 400
//  2. วันรับเข้าเลี้ยงต้องไม่ก่อนวันเกิดไก่ (เอาไก่เข้าคอกก่อนเกิดไม่ได้)
//     ทำไมเช็กที่ backend ด้วย: ถ้าหน้าแอป/เว็บลืมเช็ก หรือมีคนยิง API ตรง ๆ ก็ยังถูกกัน
//  3. บันทึก ถ้าชื่อคอกซ้ำในฟาร์มเดียวกัน -> 409
//  4. สำเร็จ -> 201 พร้อมข้อมูลคอก
func CreateCoopHandler(w http.ResponseWriter, r *http.Request) {
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

	var req models.CreateCoopRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.NameCoop == "" || req.DateAdoptAnimals.IsZero() || req.Amount == 0 || req.Birthday.IsZero() {
		log.Printf("[%s] %s - %d (Missing required fields)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing required fields: name_coop, date_adopt_animals, amount, birthday", http.StatusBadRequest)
		return
	}

	// เอาไก่เข้าคอกก่อนไก่เกิดไม่ได้ - กันไว้ที่ backend ด้วย ไม่ใช่แค่ฝั่งเว็บ/แอป
	// เพราะ client ฝั่งไหนพลาดไม่เช็คมา (หรือยิง API ตรงๆ) ก็ยังโดนกันตรงนี้
	if req.DateAdoptAnimals.Before(req.Birthday) {
		log.Printf("[%s] %s - %d (date_adopt_animals before birthday)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "วันที่รับเข้าเลี้ยงต้องไม่ก่อนวันเกิดไก่", http.StatusBadRequest)
		return
	}

	coop, err := database.CreateCoop(&req, userID)
	if err != nil {
		if database.IsDuplicateNameCoop(err) {
			log.Printf("[%s] %s - %d (Duplicate coop name)", r.Method, r.RequestURI, http.StatusConflict)
			http.Error(w, "Coop name already exists", http.StatusConflict)
			return
		}
		log.Printf("[%s] %s - %d (Failed to create coop: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to create coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	log.Printf("[%s] %s - %d ✓ Created coop ID: %d", r.Method, r.RequestURI, http.StatusCreated, coop.CoopID)
	json.NewEncoder(w).Encode(coop)
}

// GetCoopHandler (GET /api/coops?id=1) ดึงคอกเดียวพร้อมอุปกรณ์ ไข่ สุขภาพ วัคซีน
// เติมค่าเซนเซอร์ล่าสุดให้อุปกรณ์แต่ละตัว (attachLatestSensorValues ใน device.go)
// หน้าจอจะได้แสดงอุณหภูมิ/แอมโมเนียปัจจุบันของคอกได้เลย
// ไม่เจอ หรือไม่ใช่ของผู้ใช้ -> 404
func GetCoopHandler(w http.ResponseWriter, r *http.Request) {
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

	// Extract ID from query parameter
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing coop id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid coop id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	coop, err := database.GetCoopByIDForUser(id, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
		http.Error(w, "Coop not found", http.StatusNotFound)
		return
	}

	attachLatestSensorValues(coop.Devices)

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved coop ID: %d", r.Method, r.RequestURI, http.StatusOK, coop.CoopID)
	json.NewEncoder(w).Encode(coop)
}

// GetAllCoopsHandler (GET /api/coops) ดึงคอกทั้งหมดของผู้ใช้ ใช้ในหน้าหลักและหน้ารายการคอก
// แถม: เรียก RunCatchUpDeduction (ตัดสต็อกอาหารที่ค้าง) แบบ goroutine
// ทำไม: หน้านี้ถูกเปิดบ่อยที่สุด ถ้าเซิร์ฟเวอร์หลับไปตอน 18:00 น. จน cron ไม่ทำงาน
// จะได้ตัดสต็อกย้อนหลังให้ตอนผู้ใช้เปิดแอป (ไม่ต้องรอผล เพราะหน้านี้ไม่ได้แสดงสต็อก)
func GetAllCoopsHandler(w http.ResponseWriter, r *http.Request) {
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

	// เป็นหน้าที่ถูกเปิดบ่อยที่สุด (หน้าแรกของทั้งเว็บและแอป) เลยอาศัยจังหวะนี้เป็น
	// จุดสำรองเพิ่มเติมให้ RunCatchUpDeduction อีกจุด (นอกจากหน้าคลังอาหาร) - รันเป็น
	// goroutine แยก ไม่ต้องรอผลเพราะหน้านี้ไม่ได้โชว์ตัวเลขสต็อกอาหารอยู่แล้ว
	go RunCatchUpDeduction(userID)

	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch coops", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d coops", r.Method, r.RequestURI, http.StatusOK, len(coops))
	json.NewEncoder(w).Encode(coops)
}

// UpdateCoopHandler (PUT /api/coops?id=1) แก้ไขข้อมูลคอก (หน้า "แก้ไขข้อมูลคอกไก่")
//   - ถ้าส่งมาทั้งวันรับเข้าเลี้ยงและวันเกิด วันรับเข้าเลี้ยงต้องไม่ก่อนวันเกิด
//   - ไม่เจอ/ไม่ใช่ของผู้ใช้ -> 404, ชื่อซ้ำ -> 409
func UpdateCoopHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[%s] %s - %d (Missing coop id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid coop id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	var req models.UpdateCoopRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// เอาไก่เข้าคอกก่อนไก่เกิดไม่ได้ - เช็คเฉพาะตอนส่งมาทั้งคู่ (ฟอร์มแก้ไขของทั้ง
	// เว็บ/แอปส่งมาครบทั้งสองฟิลด์เสมอ ไม่ใช่ partial update)
	if !req.DateAdoptAnimals.IsZero() && !req.Birthday.IsZero() && req.DateAdoptAnimals.Before(req.Birthday) {
		log.Printf("[%s] %s - %d (date_adopt_animals before birthday)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "วันที่รับเข้าเลี้ยงต้องไม่ก่อนวันเกิดไก่", http.StatusBadRequest)
		return
	}

	coop, err := database.UpdateCoop(id, &req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		if database.IsDuplicateNameCoop(err) {
			log.Printf("[%s] %s - %d (Duplicate coop name)", r.Method, r.RequestURI, http.StatusConflict)
			http.Error(w, "Coop name already exists", http.StatusConflict)
			return
		}
		log.Printf("[%s] %s - %d (Failed to update coop: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Updated coop ID: %d", r.Method, r.RequestURI, http.StatusOK, coop.CoopID)
	json.NewEncoder(w).Encode(coop)
}

// DeleteCoopHandler (DELETE /api/coops?id=1) ลบคอกพร้อมข้อมูลทั้งหมดในคอก
// (database.DeleteCoop ทำใน Transaction) ไม่เจอ -> 404
func DeleteCoopHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[%s] %s - %d (Missing coop id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing coop id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid coop id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid coop id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteCoop(id, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to delete coop: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to delete coop", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Deleted coop ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
	json.NewEncoder(w).Encode(map[string]string{"message": "Coop deleted successfully"})
}
