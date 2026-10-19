// =====================================================================================
// ไฟล์ handlers/sensor_log.go - API เกี่ยวกับค่าเซนเซอร์ และการแจ้งเตือนการเคลื่อนไหว
// =====================================================================================
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// DeviceResponse รูปแบบข้อมูลอุปกรณ์ที่ส่งให้หน้าเว็บ Angular (ใช้ใน GetCoopDataHandler)
// ใช้ชื่อ field "status" ตัวเล็ก ให้ตรงกับที่หน้าเว็บรออ่าน
type DeviceResponse struct {
	DeviceID  int    `json:"device_id"`
	SlotIndex int    `json:"slot_index"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Status    string `json:"status"`
}

// GetMotionAlertsHandler (GET /api/motion-alerts) ส่งการตรวจจับการเคลื่อนไหว
// จากเซนเซอร์ PIR (ติดที่ประตูคอก) ในช่วง 24 ชั่วโมงที่ผ่านมา เรียงจากล่าสุด
// ใช้แจ้งเตือนว่ามีคน/สัตว์เข้าคอก ข้อมูลจาก database.GetMotionAlertsForUser (เฉพาะคอกของผู้ใช้)
func GetMotionAlertsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	rows, err := database.GetMotionAlertsForUser(userID, since)
	if err != nil {
		log.Printf("GetMotionAlertsHandler: failed to fetch motion alerts for user %d: %v", userID, err)
		http.Error(w, "Failed to fetch motion alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

// ReceiveSensorDataHandler (POST /api/sensor-logs) รับค่าเซนเซอร์ผ่าน HTTP แล้วบันทึกเป็นแถวใหม่
// เป็นช่องทางสำรอง/ทดสอบ ช่องทางหลักคือ MQTT (services/sensor_log.go)
// ไม่ต้องล็อกอิน เพราะบอร์ดไม่มี JWT ตอบ 201 พร้อม log_id ของแถวที่บันทึก
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

// GetCoopDataHandler ส่งรายการอุปกรณ์ของคอก (?id=) พร้อมสถานะ Online/Offline
// ถ้าอุปกรณ์ไม่อัปเดตเกิน 5 นาที จะบังคับเป็น "offline" ตอนตอบกลับ
// หมายเหตุ: ฟังก์ชันนี้ "ไม่ได้ลงทะเบียน" ใน routes/routes.go แล้ว (เป็นโค้ดรุ่นเก่า)
// หน้าเว็บปัจจุบันใช้ GET /api/devices แทน และสถานะ Offline ถูกตั้งโดย services.StartOfflineChecker
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

			Name:   dbDevice.Name,
			Icon:   dbDevice.Icon,
			Status: finalStatus,
		})
	}

	// 4. ห่อ JSON ส่งกลับไปให้หน้า Frontend นำไปเรนเดอร์สี
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"coop_id": int32(coopID),
		"devices": responseDevices,
	})
}
