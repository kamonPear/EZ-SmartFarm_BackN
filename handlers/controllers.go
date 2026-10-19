// =====================================================================================
// ไฟล์ handlers/controllers.go - API รับข้อมูลจากบอร์ดผ่าน HTTP (ทางสำรอง / รุ่นแรก)
// =====================================================================================
//
// ช่องทางหลักที่บอร์ดใช้ส่งข้อมูลตอนนี้คือ MQTT (services/mqtt_service.go)
// ไฟล์นี้เป็นทางเลือกแบบ HTTP สำหรับทดสอบ: แค่เช็กว่าอุปกรณ์มีอยู่ในผังและตั้งเป็น Online
//
// หมายเหตุสำหรับ handler ทุกไฟล์ในโฟลเดอร์ handlers/:
// handler = ฟังก์ชันรับ HTTP request (r) แล้วเขียนคำตอบกลับ (w) ส่วนใหญ่ทำตามแบบเดียวกัน คือ
//  1. เช็ก method (GET/POST/PUT/DELETE)
//  2. ดึง userID ของผู้ที่ล็อกอิน จาก auth.UserIDFromContext(r)
//     (middleware RequireAuth ตรวจ JWT แล้วใส่ userID ไว้ใน context ก่อนเข้ามาที่นี่)
//  3. อ่าน JSON จาก body หรือ id จาก query string (?id=)
//  4. ตรวจความถูกต้องของข้อมูล แล้วเรียกฟังก์ชันใน database/
//  5. แปลง error เป็นรหัส HTTP (ErrNotFound -> 404, ข้อมูลผิด -> 400, อื่น ๆ -> 500)
//  6. ตอบกลับเป็น JSON
package handlers // ปรับชื่อ package ตามโปรเจกต์คุณ

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"EZ-SmartFarm_BachN/models" // อย่าลืมเปลี่ยนเป็น path ของ models คุณ
	"gorm.io/gorm"
)

// HandleArduinoUpload สร้าง handler สำหรับ POST /api/sensor/upload (บอร์ดส่งข้อมูลผ่าน HTTP)
// รับ db เข้ามาแล้วคืนฟังก์ชัน handler (closure) เพื่อให้ handler ใช้ db ตัวนี้ได้
//
// ขั้นตอน:
//  1. แปลง JSON เป็น ArduinoPayload
//  2. หาอุปกรณ์จาก (coop_id, slot_index) ถ้าไม่เจอ -> 404 "ไม่มีอุปกรณ์นี้ในผัง"
//  3. ถ้าเจอ ตั้งสถานะเป็น Online และอัปเดตเวลาล่าสุด
//
// ไม่ต้องล็อกอิน (บอร์ดไม่มี JWT) และยังไม่ได้บันทึกค่าเซนเซอร์ (ทางหลักบันทึกผ่าน MQTT)
// ถูกลงทะเบียนใน: routes/routes.go
func HandleArduinoUpload(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. แปลง JSON จาก Request มาใส่ใน Struct
		var payload models.ArduinoPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// 2. ค้นหา Device ในฐานข้อมูล (ใช้ slot_index จับคู่แทนชื่อ เพราะตอนนี้ชื่อซ้ำกันได้ในคนละ slot)
		var device models.Device
		result := db.Where("coop_id = ? AND slot_index = ?", payload.CoopID, payload.SlotIndex).First(&device)

		// 3. ตรวจสอบผลลัพธ์
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				http.Error(w, "Error: Device not found in layout", http.StatusNotFound)
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// 4. ถ้าหาเจอ อัปเดตสถานะให้เป็น Online
		db.Model(&device).Updates(models.Device{
			CurrentStatus: "Online",
			LastUpdate:    time.Now(),
		})

		// ** ตรงนี้คุณสามารถเขียนโค้ดเพื่อเอา payload.SensorValue ไปบันทึกลงตาราง SensorLog ต่อได้เลย **

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Data received and checked successfully"))
	}
}
