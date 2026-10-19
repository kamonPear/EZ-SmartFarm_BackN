// =====================================================================================
// ไฟล์ handlers/farm_layout.go - API ผังฟาร์ม (รูปทรงขอบฟาร์ม และตำแหน่งคอกบนผัง)
// =====================================================================================
//
// หน้าเว็บ "จัดวางผังฟาร์ม": ผู้ใช้เลือกรูปทรงขอบฟาร์ม (วงกลม/สามเหลี่ยม/สี่เหลี่ยม)
// แล้วลากคอกไปวางบนผัง ตำแหน่งเก็บเป็นเปอร์เซ็นต์ (pos_x, pos_y) จึงแสดงถูกทุกขนาดจอ
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

// GetFarmLayoutHandler (GET /api/farm-layout) ส่งรูปทรงขอบฟาร์มที่ผู้ใช้เลือกไว้
// ข้อมูลจาก database.GetFarmLayout (ถ้ายังไม่เคยเลือก จะได้ค่าเริ่มต้น)
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

// UpdateFarmLayoutHandler (PUT /api/farm-layout) บันทึกรูปทรงขอบฟาร์ม body: {"shape": "circle"}
// ตรวจก่อนว่าเป็น circle / triangle / square เท่านั้น (models.IsValidFarmShape) ไม่ใช่ -> 400
// ทำไมต้องตรวจ: หน้าเว็บวาดได้แค่ 3 รูปนี้ ถ้าเก็บค่าอื่นลงไป ผังจะแสดงไม่ได้
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

// UpdateCoopPositionsHandler (PUT /api/coops/positions) บันทึกตำแหน่งทุกคอกบนผังในครั้งเดียว
// body: {"positions": [{"coop_id":1, "pos_x":42.5, "pos_y":10.2}, ...]}
// ส่งต่อให้ database.UpdateCoopPositions (เช็กว่าทุกคอกเป็นของผู้ใช้ ใน Transaction)
// ถ้ามีคอกไหนไม่ใช่ของผู้ใช้ -> 404 และไม่บันทึกเลยสักคอก
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
