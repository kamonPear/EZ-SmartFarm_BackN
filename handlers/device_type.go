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

type createDeviceTypeRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// defaultDeviceTypes คือชนิดอุปกรณ์มาตรฐาน 7 แบบ ให้ทุก user เป็นแถวจริงใน
// device_type ของตัวเองครั้งแรกที่เปิดดู (ดู ensureDefaultDeviceTypes) ไอคอนตรงกับ
// DEVICE_ICON_CHOICES ฝั่งเว็บ (src/app/shared/device-icon.util.ts) เป๊ะ - ตั้งใจ
// ให้เป็นแถวจริงที่แก้ไข/ลบถาวรได้เหมือนชนิดที่ผู้ใช้เพิ่มเอง ไม่ใช่ค่าฝังตายตัวที่
// ลบไม่ได้อีกต่อไป ตามที่ผู้ใช้ขอ
var defaultDeviceTypes = []struct{ Name, Icon string }{
	{"ESP 32", "data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%232E9E4F%22%20stroke-width%3D%221.7%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Crect%20x%3D%226%22%20y%3D%226%22%20width%3D%2212%22%20height%3D%2212%22%20rx%3D%221.5%22%2F%3E%3Crect%20x%3D%229.5%22%20y%3D%229.5%22%20width%3D%225%22%20height%3D%225%22%2F%3E%3Cpath%20d%3D%22M9%202v3M15%202v3M9%2019v3M15%2019v3M2%209h3M2%2015h3M19%209h3M19%2015h3%22%2F%3E%3C%2Fsvg%3E"},
	{"MQ-135", "data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%232E9E4F%22%20stroke-width%3D%221.7%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Cpath%20d%3D%22M3%208h11a3%203%200%201%200-3-3%22%2F%3E%3Cpath%20d%3D%22M3%2012h15a3%203%200%201%201-3%203%22%2F%3E%3Cpath%20d%3D%22M3%2016h8%22%2F%3E%3C%2Fsvg%3E"},
	{"PIR MOTION", "data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%232E9E4F%22%20stroke-width%3D%221.7%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Ccircle%20cx%3D%2212%22%20cy%3D%2218%22%20r%3D%221.3%22%20fill%3D%22%232E9E4F%22%20stroke%3D%22none%22%2F%3E%3Cpath%20d%3D%22M8.5%2014.5a5%205%200%200%201%207%200%22%2F%3E%3Cpath%20d%3D%22M5.5%2011.5a9%209%200%200%201%2013%200%22%2F%3E%3C%2Fsvg%3E"},
	{"DHT22", "data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%232E9E4F%22%20stroke-width%3D%221.7%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Cpath%20d%3D%22M10%2013.4V4.5a2%202%200%201%201%204%200v8.9a4%204%200%201%201-4%200Z%22%2F%3E%3Cline%20x1%3D%2212%22%20y1%3D%227%22%20x2%3D%2212%22%20y2%3D%2213.5%22%2F%3E%3C%2Fsvg%3E"},
	{"MC-38", "data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%232E9E4F%22%20stroke-width%3D%221.7%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Crect%20x%3D%222%22%20y%3D%229%22%20width%3D%227%22%20height%3D%226%22%20rx%3D%221.2%22%2F%3E%3Crect%20x%3D%2215%22%20y%3D%229%22%20width%3D%227%22%20height%3D%226%22%20rx%3D%221.2%22%2F%3E%3Cpath%20d%3D%22M9%2012h6%22%20stroke-dasharray%3D%222.2%202.2%22%2F%3E%3C%2Fsvg%3E"},
	{"พัดลม", "data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%232E9E4F%22%20stroke-width%3D%221.7%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Ccircle%20cx%3D%2212%22%20cy%3D%2212%22%20r%3D%221.4%22%20fill%3D%22%232E9E4F%22%20stroke%3D%22none%22%2F%3E%3Cpath%20d%3D%22M12%2012c0-3.2-2.2-5.2-5.2-5.2.1%203.2%202%205.2%205.2%205.2Z%22%2F%3E%3Cpath%20d%3D%22M12%2012c3.2%200%205.2-2.2%205.2-5.2-3.2.1-5.2%202-5.2%205.2Z%22%2F%3E%3Cpath%20d%3D%22M12%2012c0%203.2%202.2%205.2%205.2%205.2-.1-3.2-2-5.2-5.2-5.2Z%22%2F%3E%3Cpath%20d%3D%22M12%2012c-3.2%200-5.2%202.2-5.2%205.2%203.2-.1%205.2-2%205.2-5.2Z%22%2F%3E%3C%2Fsvg%3E"},
	{"หลอดไฟ", "data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20viewBox%3D%220%200%2024%2024%22%20fill%3D%22none%22%20stroke%3D%22%232E9E4F%22%20stroke-width%3D%221.7%22%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%3E%3Cpath%20d%3D%22M9%2018h6%22%2F%3E%3Cpath%20d%3D%22M10%2022h4%22%2F%3E%3Cpath%20d%3D%22M12%202a7%207%200%200%200-4%2012.7c.6.5%201%201.3%201%202.1V17h6v-.2c0-.8.4-1.6%201-2.1A7%207%200%200%200%2012%202Z%22%2F%3E%3C%2Fsvg%3E"},
}

// ensureDefaultDeviceTypes สร้างชนิดอุปกรณ์มาตรฐาน 7 แบบให้ userID ครั้งแรกที่เปิด
// ดูเท่านั้น (เช็คจาก User.DeviceTypesSeeded ไม่ใช่นับจำนวนแถว - ถ้า user ลบทิ้งจน
// เหลือ 0 แถวในภายหลัง ต้องไม่ seed กลับมาให้ใหม่ เพราะถือว่าตั้งใจลบถาวรแล้ว)
func ensureDefaultDeviceTypes(userID int) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		log.Printf("Failed to load user for device type seeding: %v", err)
		return
	}
	if user.DeviceTypesSeeded {
		return
	}

	rows := make([]models.DeviceType, 0, len(defaultDeviceTypes))
	for _, d := range defaultDeviceTypes {
		rows = append(rows, models.DeviceType{UserID: userID, Name: d.Name, Icon: d.Icon})
	}
	if err := database.DB.Create(&rows).Error; err != nil {
		log.Printf("Failed to seed default device types for user %d: %v", userID, err)
		return
	}
	if err := database.DB.Model(&user).Update("device_types_seeded", true).Error; err != nil {
		log.Printf("Failed to mark device types seeded for user %d: %v", userID, err)
	}
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

	ensureDefaultDeviceTypes(userID)

	var rows []models.DeviceType
	if err := database.DB.Where("user_id = ?", userID).Order("name asc").Find(&rows).Error; err != nil {
		log.Printf("Failed to fetch device types: %v", err)
		http.Error(w, "Failed to fetch device types", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(rows)
}

type updateDeviceTypeRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// UpdateDeviceTypeHandler แก้ไขชื่อ/ไอคอนของชนิดอุปกรณ์ที่เพิ่มเอง - เฉพาะของตัวเองเท่านั้น
// PUT /api/device-types?id=1
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

// DeleteDeviceTypeHandler ลบชนิดอุปกรณ์ที่เพิ่มเอง - เฉพาะของตัวเองเท่านั้น (ไม่กระทบ
// อุปกรณ์ที่ถูกวางไปแล้วในคอกต่างๆ เพราะตอนวางจะ copy ชื่อ+ไอคอนไปเก็บแยกต่างหาก
// ไม่ได้อ้างอิงกลับมาที่แถวนี้)
// DELETE /api/device-types?id=1
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
