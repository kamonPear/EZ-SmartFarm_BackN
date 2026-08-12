package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
	// 🔥 ลบ "gorm.io/gorm" ออกไปแล้ว เพราะเราใช้ผ่านแพ็กเกจ database แทน
)

// SaveCoopLayoutHandler จัดการการบันทึกตำแหน่งอุปกรณ์ทั้ง 21 ช่องจาก Angular
// POST /api/coops/layout?coop_id=1
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

	if err := database.SaveCoopLayout(coopID, payload.Slots); err != nil {
		log.Printf("[%s] %s - %d (Failed to save layout: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to save layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

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

// GetAllDevicesHandler ดึงข้อมูลอุปกรณ์ทั้งหมด (รองรับอุปกรณ์จำนวนมากแบบความเร็วสูง)
func GetAllDevicesHandler(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	var devices []models.Device

	if err := db.Find(&devices).Error; err != nil {
		http.Error(w, "Failed to fetch devices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(devices) > 0 {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// CreateDeviceHandler สร้างอุปกรณ์ใหม่
func CreateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.CreateDevice(&device); err != nil {
		http.Error(w, "Failed to create device: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

// GetDeviceHandler ดึงข้อมูลอุปกรณ์ตาม ID
func GetDeviceHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	device, err := database.GetDeviceByID(id)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

// UpdateDeviceHandler แก้ไขข้อมูลอุปกรณ์
func UpdateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.UpdateDevice(&device); err != nil {
		http.Error(w, "Failed to update device: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

// DeleteDeviceHandler ลบอุปกรณ์ตาม ID หรือตาม (CoopID + SlotIndex)
func DeleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	db := database.GetDB()
	if db == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	idStr := r.URL.Query().Get("id")
	coopIDStr := r.URL.Query().Get("coop_id")
	slotIndexStr := r.URL.Query().Get("slot_index")

	if coopIDStr != "" && slotIndexStr != "" {
		coopID, _ := strconv.Atoi(coopIDStr)
		slotIndex, _ := strconv.Atoi(slotIndexStr)

		result := db.Where("coop_id = ? AND slot_index = ?", coopID, slotIndex).Delete(&models.Device{})
		if result.Error != nil {
			http.Error(w, "Failed to delete device by slot: "+result.Error.Error(), http.StatusInternalServerError)
			return
		}
	} else if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		result := db.Delete(&models.Device{}, id)
		if result.Error != nil {
			http.Error(w, "Failed to delete device by ID: "+result.Error.Error(), http.StatusInternalServerError)
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

// GetDevicesByCoopHandler ดึงอุปกรณ์ทั้งหมดที่อยู่ใน 1 คอก (รองรับอุปกรณ์จำนวนมากแบบความเร็วสูง)
func GetDevicesByCoopHandler(w http.ResponseWriter, r *http.Request) {
	coopIDStr := r.URL.Query().Get("coop_id")
	coopID, _ := strconv.Atoi(coopIDStr)

	db := database.GetDB()
	if db == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	var devices []models.Device

	if err := db.Where("coop_id = ?", coopID).Find(&devices).Error; err != nil {
		http.Error(w, "Failed to fetch devices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(devices) > 0 {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}