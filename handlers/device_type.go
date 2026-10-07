package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

type createDeviceTypeRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// CreateDeviceTypeHandler เพิ่ม "ชนิด" อุปกรณ์ใหม่ (ชื่อ+ไอคอน) ไว้ใช้ซ้ำตอนจัดวาง
// อุปกรณ์ลงคอก - เฉพาะในฟาร์ม (user) ของตัวเองเท่านั้น
// POST /api/device-types
func CreateDeviceTypeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createDeviceTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Icon == "" {
		http.Error(w, "Missing required fields: name, icon", http.StatusBadRequest)
		return
	}

	// กันชื่อซ้ำ (ไม่สนตัวพิมพ์เล็ก-ใหญ่) เฉพาะในฟาร์มเดียวกัน - ฟาร์มอื่นตั้งชื่อซ้ำกันได้
	var existingCount int64
	if err := database.DB.Model(&models.DeviceType{}).
		Where("user_id = ? AND LOWER(name) = LOWER(?)", userID, req.Name).
		Count(&existingCount).Error; err != nil {
		log.Printf("Failed to check duplicate device type name: %v", err)
		http.Error(w, "Failed to save device type", http.StatusInternalServerError)
		return
	}
	if existingCount > 0 {
		http.Error(w, "มีชนิดอุปกรณ์ชื่อนี้อยู่แล้ว กรุณาใช้ชื่ออื่น", http.StatusConflict)
		return
	}

	deviceType := models.DeviceType{
		UserID: userID,
		Name:   req.Name,
		Icon:   req.Icon,
	}
	if err := database.DB.Create(&deviceType).Error; err != nil {
		log.Printf("Failed to save device type: %v", err)
		http.Error(w, "Failed to save device type", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(deviceType)
}

// GetDeviceTypesHandler คืนรายการชนิดอุปกรณ์ที่ผู้ใช้เพิ่มเองไว้ทั้งหมด
// GET /api/device-types
func GetDeviceTypesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var rows []models.DeviceType
	if err := database.DB.Where("user_id = ?", userID).Order("name asc").Find(&rows).Error; err != nil {
		log.Printf("Failed to fetch device types: %v", err)
		http.Error(w, "Failed to fetch device types", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(rows)
}
