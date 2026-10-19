// =====================================================================================
// ไฟล์ handlers/device.go - API อุปกรณ์ในคอก และการบันทึกผังอุปกรณ์
// =====================================================================================
//
// ข้อมูลอยู่ใน database/device_repository.go
// หลังเปลี่ยนอุปกรณ์ในคอก จะส่งเลขช่องใหม่ให้บอร์ดทาง MQTT (services.PublishSlotsForCoop)
// GET /api/devices แยกที่ handleDevicesGet ใน routes.go:
//
//	?id= -> GetDeviceHandler, ?coop_id= -> GetDevicesByCoopHandler, ไม่มี -> GetAllDevicesHandler
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
	"EZ-SmartFarm_BachN/services"
	// 🔥 ลบ "gorm.io/gorm" ออกไปแล้ว เพราะเราใช้ผ่านแพ็กเกจ database แทน
)

// SaveCoopLayoutHandler (POST /api/coops/layout?coop_id=1) บันทึกผังอุปกรณ์ทั้งคอก
// เรียกจากปุ่ม "บันทึกการจัดวาง" ในหน้าจัดวางระบบเซนเซอร์ (เว็บ Angular)
// body = รายการช่องทั้งหมด (models.LayoutPayload) แต่ละช่องมีชื่อ ไอคอน ตำแหน่ง
//  1. ตั้ง header CORS และตอบ OPTIONS (preflight ของเบราว์เซอร์) ทันที
//  2. อ่าน coop_id และ body แล้วให้ database.SaveCoopLayout ซิงก์อุปกรณ์ใน Transaction
//     คอกไม่ใช่ของผู้ใช้ -> 404
//  3. ส่งเลขช่องใหม่ให้บอร์ดของคอกนี้ใน goroutine (ไม่ให้หน้าเว็บรอ MQTT)
func SaveCoopLayoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

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

	var payload models.LayoutPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body: %v)", r.Method, r.RequestURI, http.StatusBadRequest, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.SaveCoopLayout(coopID, payload.Slots, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Coop not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to save layout: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to save layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// แจ้งบอร์ดของคอกนี้ว่าอุปกรณ์แต่ละชนิดอยู่ช่องไหน (ไม่ให้ response ช้าตาม MQTT)
	go services.PublishSlotsForCoop(int32(coopID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ บันทึก Layout สำเร็จสำหรับคอก ID: %d", r.Method, r.RequestURI, http.StatusOK, coopID)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "บันทึกการจัดวางอุปกรณ์สำเร็จ",
	})
}

// ==========================================
// 🌟 ฟังก์ชัน CRUD สำหรับอุปกรณ์ (Device)
// ==========================================

// attachLatestSensorValues เติม "ค่าล่าสุดจากเซนเซอร์" ลงในอุปกรณ์แต่ละตัว (field Value)
// ใช้ query เดียวดึงแถวล่าสุด (MAX(log_id)) ของทุกอุปกรณ์พร้อมกัน แล้วจับคู่ด้วย map
// ทำไมไม่ query ทีละตัว: ถ้ามี 20 อุปกรณ์จะต้องยิง 20 ครั้ง (ปัญหา N+1) ช้ากว่ามาก
// อุปกรณ์ที่ยังไม่เคยส่งค่า จะได้ Value = nil (หน้าจอแสดง "-")
// แก้ slice ที่ส่งเข้ามาโดยตรง (ไม่ต้องคืนค่า)
// ถูกเรียกจาก: GetAllDevicesHandler, GetDevicesByCoopHandler ในไฟล์นี้ และ GetCoopHandler ใน coop.go
func attachLatestSensorValues(devices []models.Device) {
	db := database.GetDB()
	if db == nil || len(devices) == 0 {
		return
	}

	var deviceIDs []int32
	for _, d := range devices {
		deviceIDs = append(deviceIDs, d.DeviceID)
	}

	var latestLogs []models.SensorLog
	db.Where("log_id IN (SELECT MAX(log_id) FROM sensor_log WHERE device_id IN ? GROUP BY device_id)", deviceIDs).Find(&latestLogs)

	logMap := make(map[int32]float64)
	for _, log := range latestLogs {
		logMap[log.DeviceID] = log.Value
	}

	for i, d := range devices {
		if val, ok := logMap[d.DeviceID]; ok {
			v := val
			devices[i].Value = &v
		} else {
			devices[i].Value = nil
		}
	}
}

// GetAllDevicesHandler (GET /api/devices) ดึงอุปกรณ์ทุกตัวในทุกคอกของผู้ใช้ พร้อมค่าล่าสุด
// ใช้ในหน้าสถานะการทำงานของอุปกรณ์
func GetAllDevicesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	devices, err := database.GetAllDevices(userID)
	if err != nil {
		http.Error(w, "Failed to fetch devices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	attachLatestSensorValues(devices)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// CreateDeviceHandler (POST /api/devices) เพิ่มอุปกรณ์ทีละตัว
// คอกไม่ใช่ของผู้ใช้ -> 404 สำเร็จ -> 201
func CreateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.CreateDevice(&device, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to create device: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

// GetDeviceHandler (GET /api/devices?id=) ดึงอุปกรณ์ตัวเดียว ไม่เจอ/ไม่ใช่ของผู้ใช้ -> 404
func GetDeviceHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	device, err := database.GetDeviceByID(id, userID)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

// UpdateDeviceHandler (PUT /api/devices) แก้ไขข้อมูลอุปกรณ์ (id อยู่ใน body)
// ไม่เจอ -> 404
func UpdateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.UpdateDevice(&device, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Device not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update device: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

// DeleteDeviceHandler (DELETE /api/devices) ลบอุปกรณ์ได้ 2 แบบ
//   - ?coop_id=&slot_index= : ลบตามตำแหน่งในคอก (หน้าจัดวางใช้แบบนี้ เพราะรู้แค่ตำแหน่งช่อง)
//     แล้วส่งเลขช่องใหม่ให้บอร์ดทาง MQTT
//   - ?id= : ลบตาม id ของอุปกรณ์
//     ไม่ส่งทั้งสองแบบ -> 400
func DeleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	coopIDStr := r.URL.Query().Get("coop_id")
	slotIndexStr := r.URL.Query().Get("slot_index")

	if coopIDStr != "" && slotIndexStr != "" {
		coopID, _ := strconv.Atoi(coopIDStr)
		slotIndex, _ := strconv.Atoi(slotIndexStr)

		if err := database.DeleteDeviceBySlot(coopID, slotIndex, userID); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				http.Error(w, "Coop not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to delete device by slot: "+err.Error(), http.StatusInternalServerError)
			return
		}
		go services.PublishSlotsForCoop(int32(coopID))
	} else if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		if err := database.DeleteDevice(id, userID); err != nil {
			if errors.Is(err, database.ErrNotFound) {
				http.Error(w, "Device not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to delete device by ID: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Missing required parameters (id OR coop_id + slot_index)", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Deleted successfully"})
}

// GetDevicesByCoopHandler (GET /api/devices?coop_id=) ดึงอุปกรณ์ของคอกเดียว พร้อมค่าล่าสุด
// ใช้ในหน้าสถานะอุปกรณ์รายคอก และโหลดผังเดิมในหน้าจัดวาง คอกไม่ใช่ของผู้ใช้ -> 404
func GetDevicesByCoopHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	coopIDStr := r.URL.Query().Get("coop_id")
	coopID, _ := strconv.Atoi(coopIDStr)

	devices, err := database.GetDevicesByCoopID(coopID, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch devices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	attachLatestSensorValues(devices)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}
