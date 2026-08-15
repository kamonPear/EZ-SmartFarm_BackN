package handlers // ปรับชื่อ package ตามโปรเจกต์คุณ

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"EZ-SmartFarm_BachN/models" // อย่าลืมเปลี่ยนเป็น path ของ models คุณ
	"gorm.io/gorm"
)

// ฟังก์ชันสำหรับรับข้อมูลจาก Arduino
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