package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// SaveCoopLayoutHandler จัดการการบันทึกตำแหน่งอุปกรณ์ทั้ง 21 ช่องจาก Angular
// POST /api/coops/layout?coop_id=1
func SaveCoopLayoutHandler(w http.ResponseWriter, r *http.Request) {
	// 🌟 บล๊อกที่ 1: ปลดล็อก CORS ให้ Angular สามารถยิงข้ามพอร์ตมาหา Go ได้
	w.Header().Set("Access-Control-Allow-Origin", "*") 
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// 🌟 บล๊อกที่ 2: แก้ปัญหา 404 Preflight (ตอบกลับคำขอ OPTIONS ของบราวเซอร์ทันที)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 1. ตรวจสอบ Method POST
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. รับค่า coop_id จาก URL
	coopIDStr := r.URL.Query().Get("coop_id")
	if coopIDStr == "" {
		log.Printf("[%s] %s - %d (Missing coop_id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing coop_id parameter", http.StatusBadRequest)
		return
	}

	coopID, err := strconv.Atoi(coopIDStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid coop_id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid coop_id", http.StatusBadRequest)
		return
	}

	// 3. แปลง JSON Body เป็น Struct (ย้ายไปดึงมาจากแพ็กเกจ models แทนเพื่อไม่ให้โค้ดวนลูป)
	var payload models.LayoutPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body: %v)", r.Method, r.RequestURI, http.StatusBadRequest, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 🌟 บล๊อกที่ 3: สั่งบันทึกข้อมูลลงฐานข้อมูลจริงผ่านแพ็กเกจ database
	if err := database.SaveCoopLayout(coopID, payload.Slots); err != nil {
		log.Printf("[%s] %s - %d (Failed to save layout: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to save layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. ส่งผลลัพธ์กลับไปให้ Angular เมื่อบันทึกสำเร็จ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ บันทึก Layout สำเร็จสำหรับคอก ID: %d", r.Method, r.RequestURI, http.StatusOK, coopID)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "บันทึกการจัดวางอุปกรณ์สำเร็จ",
	})
}