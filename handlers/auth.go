// =====================================================================================
// ไฟล์ handlers/auth.go - API เข้าสู่ระบบ / สร้างบัญชี / ลบบัญชี / ดูข้อมูลตัวเอง
// =====================================================================================
//
// ภาพรวมการยืนยันตัวตน (authentication) ของระบบ:
//  1. ผู้ใช้ส่ง username + password มาที่ LoginHandler
//  2. ถ้าถูก ระบบออก JWT (ตั๋วดิจิทัล) ให้ (auth/jwt.go -> GenerateToken)
//  3. แอป/เว็บเก็บ token ไว้ แล้วแนบมาทุก request ใน header "Authorization: Bearer <token>"
//  4. middleware RequireAuth (auth/middleware.go) ตรวจ token แล้วใส่ userID ไว้ให้ handler ใช้
//
// การสร้าง/ลบบัญชี ต้องใช้รหัสผู้ดูแล (X-Admin-Key) ผู้ใช้ทั่วไปสมัครเองไม่ได้
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// writeJSON ตัวช่วยตอบกลับเป็น JSON: ตั้ง Content-Type, เขียนรหัส HTTP, แปลง body เป็น JSON
// เขียนเป็นฟังก์ชันไว้เพื่อไม่ต้องเขียน 3 บรรทัดนี้ซ้ำทุก handler
// ใช้ใน: handlers/auth.go, data_purge.go และไฟล์อื่น ๆ ใน handlers
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// writeJSONErr ตอบ error เป็น JSON รูปแบบ {"error": "ข้อความ"}
// แอปและเว็บจะอ่าน field error ไปแสดงให้ผู้ใช้ได้เหมือนกันทุก API
func writeJSONErr(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// LoginHandler (POST /api/auth/login ไม่ต้องล็อกอินก่อน) ตรวจชื่อผู้ใช้/รหัสผ่าน แล้วออก JWT
//
// ขั้นตอน:
//  1. ตัดช่องว่างหัวท้าย และต้องกรอกทั้งสองช่อง ไม่งั้น 400
//  2. หาผู้ใช้จากชื่อ (database.GetUserByUsername)
//  3. ตรวจ 3 อย่าง ถ้าข้อใดไม่ผ่าน ตอบ 401 "ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง" ข้อความเดียวกันหมด
//     - ไม่มีผู้ใช้นี้
//     - ชื่อตรงกันแบบตัวพิมพ์เล็ก/ใหญ่ทุกตัวหรือไม่ (MySQL เทียบแบบไม่สนตัวพิมพ์
//     "Kkk" จึงหาเจอบัญชี "kkk" ต้องเทียบซ้ำใน Go)
//     - รหัสผ่านตรงกับ hash ที่เก็บไว้ไหม (auth.CheckPassword ใช้ bcrypt)
//     ทำไมไม่บอกว่าผิดตรงไหน: กันคนร้ายเดาว่าชื่อผู้ใช้ไหนมีอยู่จริงในระบบ
//  4. ออก token (auth.GenerateToken) แล้วส่งกลับพร้อม id และชื่อผู้ใช้
//
// เรียกจาก: หน้าเข้าสู่ระบบของแอปและเว็บ
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		writeJSONErr(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := database.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("LoginHandler: error fetching user: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Generic 401 for "no such user", "wrong password", and "username matched only
	// because the username column's collation is case-insensitive" - never reveal
	// which. MySQL's default collation (utf8mb4_general_ci) means `WHERE username = ?`
	// above already matched "Kkk" against a stored "kkk" - add an exact Go-side
	// comparison so login genuinely requires the right case, not just the right
	// letters (confirmed live: logging in as "Kkk" succeeded against account "kkk").
	if user == nil || user.Username != req.Username || !auth.CheckPassword(user.Password, req.Password) {
		writeJSONErr(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		log.Printf("LoginHandler: error generating token: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// RegisterHandler (POST /api/auth/register ต้องมี X-Admin-Key) สร้างบัญชีผู้ใช้ใหม่
//  1. ต้องกรอกชื่อผู้ใช้และรหัสผ่าน
//  2. ชื่อซ้ำ -> 409 Conflict
//  3. hash รหัสผ่านด้วย bcrypt (auth.HashPassword) ก่อนเก็บ ไม่เก็บรหัสผ่านจริงลง DB
//  4. บันทึกผู้ใช้ ตอบ 201 พร้อม id และชื่อ (ไม่ส่งรหัสผ่านกลับ)
//
// ทำไมต้องใช้รหัสผู้ดูแล: ระบบให้เจ้าของฟาร์มที่ได้รับอนุญาตเท่านั้นใช้งาน ไม่เปิดให้ใครก็สมัครได้
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)

	if req.Username == "" || req.Password == "" {
		writeJSONErr(w, http.StatusBadRequest, "username and password are required")
		return
	}

	exists, err := database.UsernameExists(req.Username)
	if err != nil {
		log.Printf("RegisterHandler: error checking username: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if exists {
		writeJSONErr(w, http.StatusConflict, "username already taken")
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("RegisterHandler: error hashing password: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	user := &models.User{
		Username: req.Username,
		Password: hashed,
	}
	if err := database.CreateUser(user); err != nil {
		log.Printf("RegisterHandler: error creating user: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	log.Printf("✓ Registered new user %q", user.Username)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	})
}

// DeleteUserHandler (DELETE /api/auth/users?id=<id> ต้องมี X-Admin-Key)
// ลบบัญชีและข้อมูลทั้งหมดของบัญชีนั้นถาวร (คอก ข้อมูลในคอก สต็อกอาหาร ค่าตั้งค่า ฯลฯ)
// ทำไมใช้ X-Admin-Key แทน token ของผู้ใช้: เป็นงานของผู้ดูแล และลบได้แม้ลืมรหัสผ่านบัญชีนั้นแล้ว
// ไม่เจอผู้ใช้ -> 404 / การลบจริงทำใน database.DeleteUser (Transaction)
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSONErr(w, http.StatusBadRequest, "missing id parameter")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	if _, err := database.GetUserByID(id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			writeJSONErr(w, http.StatusNotFound, "user not found")
			return
		}
		log.Printf("DeleteUserHandler: error looking up user %d: %v", id, err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := database.DeleteUser(id); err != nil {
		log.Printf("DeleteUserHandler: error deleting user %d: %v", id, err)
		writeJSONErr(w, http.StatusInternalServerError, "internal server error")
		return
	}

	log.Printf("✓ Deleted user %d and all owned data", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "user and all owned data deleted"})
}

// MeHandler (GET /api/auth/me ต้องล็อกอิน) ส่งข้อมูลของผู้ที่ล็อกอินอยู่ (id, username)
// แอป/เว็บใช้ตรวจว่า token ที่เก็บไว้ยังใช้ได้ไหม และเอาชื่อผู้ใช้ไปแสดง
// ถ้าบัญชีถูกลบไปแล้ว (token ยังไม่หมดอายุแต่ไม่มีผู้ใช้) ตอบ 401 ให้ออกจากระบบ
func MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		writeJSONErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := database.GetUserByID(userID)
	if err != nil {
		writeJSONErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	})
}
