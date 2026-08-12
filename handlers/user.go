// handlers/user.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

func Login(w http.ResponseWriter, r *http.Request) {
	// 1. อนุญาตให้เรียกใช้งานข้ามโดเมนได้ (CORS) เผื่อใช้กับแอปหรือเว็บ
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 2. ตรวจสอบว่าต้องส่งมาเป็น POST เท่านั้น
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 3. แปลงข้อมูล JSON ที่รับมาจาก Postman หรือ Flutter
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "รูปแบบข้อมูลไม่ถูกต้อง"})
		return
	}

	// ลบช่องว่างหน้าและหลัง (กันผู้ใช้เผลอพิมพ์เว้นวรรค)
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	// 4. ไปค้นหาใน Database
	user, err := database.GetUserByUsername(req.Username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "ระบบฐานข้อมูลขัดข้อง"})
		return
	}

	// 5. ถ้าไม่พบผู้ใช้ หรือ รหัสผ่านไม่ตรง
	if user == nil || user.Password != req.Password {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง"})
		return
	}

	// 6. ถ้าถูกต้อง ส่งข้อความตอบกลับ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "เข้าสู่ระบบสำเร็จ",
		"user_id": user.ID,
	})
}