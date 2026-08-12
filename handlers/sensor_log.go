package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// สร้าง Struct จำลองข้อมูลที่จะส่งไปให้ Angular
// (สังเกตว่าเราใช้ `status` ตัวเล็ก เพื่อให้ Angular ที่เขียนดักไว้จับคู่ได้ทันที)
type DeviceResponse struct {
	DeviceID  int    `json:"device_id"`
	SlotIndex int    `json:"slot_index"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Status    string `json:"status"`
}

// ReceiveSensorDataHandler handles receiving sensor data from MQTT or IoT devices
func ReceiveSensorDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var sensorLog models.SensorLog
	err := json.NewDecoder(r.Body).Decode(&sensorLog)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create the sensor log in the database
	err = database.CreateSensorLog(&sensorLog)
	if err != nil {
		http.Error(w, "Failed to save sensor data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Sensor data received successfully",
		"log_id":  sensorLog.LogID,
	})
}

func GetCoopDataHandler(w http.ResponseWriter, r *http.Request) {
	// 1. รับค่า coop id จาก URL แล้วแปลงเป็นตัวเลข (int)
	coopIDStr := r.URL.Query().Get("id")
	coopID, err := strconv.Atoi(coopIDStr)
	if err != nil {
		http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
		return
	}

	// 2. ดึงข้อมูลจากฐานข้อมูลผ่านฟังก์ชัน GORM ที่เพิ่งสร้างใหม่
	devicesFromDB, err := database.GetDevicesByCoop(coopID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var responseDevices []DeviceResponse

	// 3. วนลูปจัดการสถานะ (ออนไลน์/ออฟไลน์)
	for _, dbDevice := range devicesFromDB {
		// ใช้ค่าสถานะล่าสุดจากฐานข้อมูลเป็นหลักก่อน
		finalStatus := dbDevice.CurrentStatus

		// นำเวลาปัจจุบัน มาลบกับ เวลาล่าสุดที่บันทึกในฐานข้อมูล
		timeDiff := time.Since(dbDevice.LastUpdate)

		// ถ้าระบบหลังบ้านพบว่าเวลาผ่านไปเกิน 5 นาทีแล้วไม่มีการอัปเดต
		// ให้บังคับเปลี่ยนสถานะเป็น offline ส่งไปให้หน้าเว็บทันที
		if timeDiff.Minutes() > 5 {
			finalStatus = "offline"
		}

		responseDevices = append(responseDevices, DeviceResponse{
		
			Name:      dbDevice.Name,
			Icon:      dbDevice.Icon,
			Status:    finalStatus,
		})
	}

	// 4. ห่อ JSON ส่งกลับไปให้หน้า Frontend นำไปเรนเดอร์สี
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"coop_id": int32(coopID),
		"devices": responseDevices,
	})
}
