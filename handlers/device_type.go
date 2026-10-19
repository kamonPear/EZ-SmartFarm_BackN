// =====================================================================================
// ไฟล์ handlers/device_type.go - API "ชนิดอุปกรณ์" (แม่แบบชื่อ+ไอคอน ไว้ลากไปวางในคอก)
// =====================================================================================
//
// ในหน้าจัดวางระบบเซนเซอร์ ด้านข้างมีรายการชนิดอุปกรณ์ (เช่น DHT22, MQ-135, พัดลม)
// ผู้ใช้เพิ่มชนิดเองได้ (ชื่อ + ไอคอน SVG) แล้วลากไปวางในช่องของคอก
// ตาราง device_type แยกตามผู้ใช้ (user_id) ไฟล์นี้คุยกับ database.DB โดยตรง เพราะ query สั้นและใช้ที่เดียว
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// createDeviceTypeRequest body ตอนเพิ่มชนิดอุปกรณ์: ชื่อ และไอคอน (SVG data URI)
type createDeviceTypeRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// CreateDeviceTypeHandler (POST /api/device-types) เพิ่มชนิดอุปกรณ์ใหม่ในฟาร์มของผู้ใช้
//  1. ต้องมีทั้งชื่อและไอคอน ไม่งั้น 400
//  2. กันชื่อซ้ำในฟาร์มเดียวกัน แบบไม่สนตัวพิมพ์ (LOWER(name) = LOWER(?)) ซ้ำ -> 409
//     ฟาร์มอื่นตั้งชื่อเดียวกันได้ เพราะเช็กเฉพาะ user_id ของตัวเอง
//  3. บันทึก ตอบ 201 พร้อมข้อมูลที่บันทึก
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

// GetDeviceTypesHandler (GET /api/device-types) ดึงชนิดอุปกรณ์ทั้งหมดของผู้ใช้ เรียงตามชื่อ
// ผู้ใช้ใหม่จะได้รายการว่าง ไม่มีชนิดตั้งต้นให้ ต้องเพิ่มเอง
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

// updateDeviceTypeRequest body ตอนแก้ไขชนิดอุปกรณ์: ชื่อใหม่ และไอคอนใหม่
type updateDeviceTypeRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// UpdateDeviceTypeHandler (PUT /api/device-types?id=1) แก้ชื่อ/ไอคอนของชนิดอุปกรณ์
//  1. หาแถวด้วย id และ user_id (เป็นของตัวเองเท่านั้น) ไม่เจอ -> 404
//  2. กันชื่อซ้ำกับชนิดอื่นในฟาร์ม (ไม่นับตัวเอง: id != ?) ซ้ำ -> 409
//  3. บันทึกด้วย DB.Save
func UpdateDeviceTypeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid device type id", http.StatusBadRequest)
		return
	}

	var req updateDeviceTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Icon == "" {
		http.Error(w, "Missing required fields: name, icon", http.StatusBadRequest)
		return
	}

	var deviceType models.DeviceType
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&deviceType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Device type not found", http.StatusNotFound)
			return
		}
		log.Printf("Failed to fetch device type: %v", err)
		http.Error(w, "Failed to update device type", http.StatusInternalServerError)
		return
	}

	// กันชื่อซ้ำกับชนิดอื่น (ไม่นับตัวเอง) เฉพาะในฟาร์มเดียวกัน
	var existingCount int64
	if err := database.DB.Model(&models.DeviceType{}).
		Where("user_id = ? AND id != ? AND LOWER(name) = LOWER(?)", userID, id, req.Name).
		Count(&existingCount).Error; err != nil {
		log.Printf("Failed to check duplicate device type name: %v", err)
		http.Error(w, "Failed to update device type", http.StatusInternalServerError)
		return
	}
	if existingCount > 0 {
		http.Error(w, "มีชนิดอุปกรณ์ชื่อนี้อยู่แล้ว กรุณาใช้ชื่ออื่น", http.StatusConflict)
		return
	}

	deviceType.Name = req.Name
	deviceType.Icon = req.Icon
	if err := database.DB.Save(&deviceType).Error; err != nil {
		log.Printf("Failed to update device type: %v", err)
		http.Error(w, "Failed to update device type", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(deviceType)
}

// DeleteDeviceTypeHandler (DELETE /api/device-types?id=1) ลบชนิดอุปกรณ์ของผู้ใช้
// ลบด้วยเงื่อนไข id + user_id ถ้าไม่มีแถวถูกลบ (RowsAffected = 0) = ไม่เจอ/ไม่ใช่ของผู้ใช้ -> 404
// ไม่กระทบอุปกรณ์ที่วางในคอกไปแล้ว เพราะตอนวางคัดลอกชื่อและไอคอนไปเก็บในตาราง device เอง
// ไม่ได้อ้างอิงกลับมาที่แถวนี้ สำเร็จตอบ 204 (ไม่มีเนื้อหา)
func DeleteDeviceTypeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid device type id", http.StatusBadRequest)
		return
	}

	res := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.DeviceType{})
	if res.Error != nil {
		log.Printf("Failed to delete device type: %v", res.Error)
		http.Error(w, "Failed to delete device type", http.StatusInternalServerError)
		return
	}
	if res.RowsAffected == 0 {
		http.Error(w, "Device type not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
