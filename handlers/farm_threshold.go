// =====================================================================================
// ไฟล์ handlers/farm_threshold.go - API ค่ามาตรฐานอุณหภูมิ/แอมโมเนียของฟาร์ม
// =====================================================================================
//
// ผู้ใช้ 1 คนมีค่ามาตรฐาน 1 ชุด (ใช้ทุกคอก) หน้าเว็บใช้แสดง/เปรียบเทียบกับค่าจริงจากเซนเซอร์
// และส่งให้บอร์ดทาง MQTT (ระบบเปิด/ปิดพัดลมหรือหลอดไฟอัตโนมัติตามค่านี้ยังไม่ได้ทำ)
// ข้อมูลอยู่ใน database/farm_threshold_repository.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
	"EZ-SmartFarm_BachN/services"
)

// GetFarmThresholdHandler (GET /api/farm-threshold) ส่งค่ามาตรฐานของผู้ใช้กลับไป
// ถ้ายังไม่เคยตั้ง ฝั่ง database จะสร้างค่าเริ่มต้นให้
// เรียกจาก: หน้าเว็บส่วนสถานะ/ค่ามาตรฐานของฟาร์ม
func GetFarmThresholdHandler(w http.ResponseWriter, r *http.Request) {
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

	threshold, err := database.GetFarmThreshold(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch farm threshold: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch farm threshold", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(threshold)
}

// UpdateFarmThresholdHandler (PUT /api/farm-threshold) บันทึกค่ามาตรฐานใหม่
// body: {"temperature": 25, "ammonia": 35}
// หลังบันทึกสำเร็จ ส่งค่าใหม่ไปให้บอร์ดทุกคอกทาง MQTT (services.PublishThresholdsForUser)
// ทำใน goroutine (go ...) เพื่อตอบหน้าเว็บทันที ไม่ต้องรอ broker ตอบกลับ
func UpdateFarmThresholdHandler(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateFarmThresholdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	threshold, err := database.UpdateFarmThreshold(&req, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to update farm threshold: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update farm threshold", http.StatusInternalServerError)
		return
	}

	// ส่งค่าใหม่ไปให้ Arduino ทุกคอกของผู้ใช้ (retained) ทำใน goroutine ไม่ให้หน้าเว็บรอ broker
	go services.PublishThresholdsForUser(userID)

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Farm threshold set to %.1f°C / %.1f ppm", r.Method, r.RequestURI, http.StatusOK, req.Temperature, req.Ammonia)
	json.NewEncoder(w).Encode(threshold)
}
