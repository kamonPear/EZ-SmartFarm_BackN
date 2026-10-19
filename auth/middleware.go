// =====================================================================================
// ไฟล์ auth/middleware.go - "ยาม" ที่ตรวจสิทธิ์ก่อนให้คำขอเข้าถึง handler
// =====================================================================================
//
// middleware คือฟังก์ชันที่ "ครอบ" handler อีกชั้น ทำงานก่อน handler จริง ถ้าตรวจไม่ผ่านจะตอบ
// error แล้วจบเลย handler จริงไม่ได้ทำงาน ถ้าผ่านจึงเรียก next(w, r) ต่อ
//
// ในไฟล์นี้มียาม 3 แบบ (เลือกใช้ใน routes/routes.go):
//  1. RequireAuth            - ผู้ใช้ทั่วไปที่ login แล้ว (ตรวจ JWT) ใช้กับ API เกือบทั้งหมด
//  2. RequireAdminKey        - ผู้ดูแลระบบ (ตรวจรหัส X-Admin-Key) ใช้กับสร้าง/ลบบัญชีผู้ใช้
//  3. RequireScheduledTaskKey - บริการปลุกจากภายนอก (cron-job.org) ใช้กับตัดสต็อกอาหารอัตโนมัติ
// และมีตัวช่วยดึง user id / username ที่ยามฝากไว้ให้ handler ใช้ต่อ
package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

// adminKeyWarn ใช้ sync.Once ให้ข้อความเตือน "ยังไม่ได้ตั้ง ADMIN_API_KEY" ขึ้นใน log แค่ครั้งเดียว
// ไม่ขึ้นซ้ำทุกครั้งที่มีคนเรียก API (log จะได้ไม่รก)
var adminKeyWarn sync.Once

// taskKeyWarn เหมือน adminKeyWarn แต่ใช้กับคำเตือนเรื่อง SCHEDULED_TASK_KEY
var taskKeyWarn sync.Once

// contextKey เป็นชนิดข้อมูลส่วนตัวที่ใช้เป็น "ชื่อช่อง" เก็บของใน context ของ request
//
// ทำไมไม่ใช้ string ธรรมดาเช่น "userID": ถ้าแพ็กเกจอื่นใช้ชื่อเดียวกันจะเขียนทับกันได้
// การสร้างชนิดเฉพาะ (contextKey) รับประกันว่าไม่ชนกับใคร (เป็นแนวปฏิบัติที่ Go แนะนำ)
// ค่าคงที่ด้านล่าง userIDContextKey / usernameContextKey ใช้ iota = ให้เลข 0, 1 อัตโนมัติ
type contextKey int

const (
	userIDContextKey contextKey = iota
	usernameContextKey
)

// writeJSONError ตอบ error กลับเป็น JSON รูปแบบเดียวกันทั้งระบบ: {"error": "ข้อความ"}
// พร้อมตั้ง status code (401 ไม่ได้ login / 403 ไม่มีสิทธิ์)
// หน้าบ้านจึงอ่าน error ได้แบบเดียวกันทุก API
// ถูกเรียกจาก: ยามทั้ง 3 ตัวในไฟล์นี้
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// RequireAuth = ยามตรวจ JWT ครอบ API ที่ต้อง login (ประมาณ 59 เส้นใน routes/routes.go)
//
// ขั้นตอน:
//  1. อ่าน header Authorization ถ้าไม่มี -> 401
//  2. ต้องอยู่ในรูป "Bearer <token>" (เทียบคำว่า Bearer แบบไม่สนตัวพิมพ์เล็กใหญ่) ถ้าผิดรูป -> 401
//  3. ParseToken (auth/jwt.go) ตรวจลายเซ็นและวันหมดอายุ ถ้าไม่ผ่าน -> 401
//  4. claims.UserID() แปลง user id เป็นตัวเลข ถ้าแปลงไม่ได้ -> 401
//  5. ฝาก userID และ username ไว้ใน context ของ request (context.WithValue)
//  6. เรียก next(...) ให้ handler จริงทำงาน โดยส่ง request ที่มี context ใหม่ไป
//
// ทำไมฝากใน context: handler จะเรียก auth.UserIDFromContext(r) แล้วรู้ทันทีว่าใครเรียก
// โดยไม่ต้องถอด token ซ้ำ และเอา user id ไปกรองข้อมูลให้เห็นเฉพาะฟาร์มของตัวเอง
// ทุกกรณีที่ไม่ผ่านตอบข้อความเดียวกันว่า unauthorized เพื่อไม่บอกใบ้ผู้ไม่หวังดีว่าผิดตรงไหน
// หน้าบ้านเห็น 401 จะเด้งกลับไปหน้า Login
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		claims, err := ParseToken(strings.TrimSpace(parts[1]))
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := claims.UserID()
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, userIDContextKey, userID)
		ctx = context.WithValue(ctx, usernameContextKey, claims.Username)

		next(w, r.WithContext(ctx))
	}
}

// RequireAdminKey = ยามสำหรับ API สร้าง/ลบบัญชีผู้ใช้ (/api/auth/register, DELETE /api/auth/users)
//
// ทำไมต้องมี: ระบบไม่มีหน้าสมัครสมาชิก และไม่มีบทบาท (role) ทุกบัญชีเท่ากันหมด
// ถ้าเปิด register ไว้เฉย ๆ ใครเจอ URL ก็สร้างบัญชีเองได้ จึงให้เฉพาะคนที่ถือรหัส
// ADMIN_API_KEY (ตั้งไว้ใน environment) เป็นคนสร้างบัญชีให้เจ้าของฟาร์ม
//
// ขั้นตอน:
//  1. ถ้าเซิร์ฟเวอร์ไม่ได้ตั้ง ADMIN_API_KEY -> ปฏิเสธทุกคน 403 ("fail closed" ปลอดภัยไว้ก่อน)
//  2. อ่าน header X-Admin-Key ที่ส่งมา
//  3. เทียบด้วย subtle.ConstantTimeCompare (ใช้เวลาเท่ากันไม่ว่าจะผิดตัวที่เท่าไร)
//     กันการโจมตีแบบจับเวลาเพื่อเดารหัสทีละตัวอักษร (timing attack)
//  4. ตรงกัน -> next(w, r) / ไม่ตรง -> 403 forbidden
func RequireAdminKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("ADMIN_API_KEY")
		if expected == "" {
			// Fail closed: an unset key must not silently leave the endpoint open.
			adminKeyWarn.Do(func() {
				log.Println("WARNING: ADMIN_API_KEY is not set - /api/auth/register is disabled until it is.")
			})
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		provided := r.Header.Get("X-Admin-Key")
		// Constant-time compare so a wrong key can't be recovered by timing the response.
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		next(w, r)
	}
}

// RequireScheduledTaskKey = ยามสำหรับ API ที่ "เครื่อง" เรียก ไม่ใช่คน
// ใช้กับ POST /api/system/food-deduction/run (ตัดสต็อกอาหารประจำวัน)
//
// ทำไมต้องมี: Render แบบฟรีจะปิดเซิร์ฟเวอร์เมื่อไม่มีคนใช้ cron ภายใน (scheduler) อาจไม่ได้รันตอน 06:00 น.
// จึงให้บริการภายนอกเช่น cron-job.org ยิงมาปลุกและสั่งตัดสต็อกแทน บริการนั้นไม่มี JWT
// จึงใช้รหัสลับ SCHEDULED_TASK_KEY แทน
//
// ขั้นตอน:
//  1. ไม่ได้ตั้ง SCHEDULED_TASK_KEY -> 403 ทุกคน (fail closed)
//  2. อ่านรหัสจาก header X-Task-Key หรือถ้าไม่มี อ่านจาก ?key= ใน URL
//     (บริการ cron ฟรีบางเจ้าตั้ง header เองไม่ได้ ใส่ได้แค่ URL)
//  3. เทียบแบบ constant-time เหมือน RequireAdminKey ตรงกันจึงให้ทำงานต่อ
func RequireScheduledTaskKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("SCHEDULED_TASK_KEY")
		if expected == "" {
			// Fail closed: an unset key must not silently leave the endpoint open.
			taskKeyWarn.Do(func() {
				log.Println("WARNING: SCHEDULED_TASK_KEY is not set - the scheduled food-deduction endpoint is disabled until it is.")
			})
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		provided := r.Header.Get("X-Task-Key")
		if provided == "" {
			provided = r.URL.Query().Get("key")
		}
		// Constant-time compare so a wrong key can't be recovered by timing the response.
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			writeJSONError(w, http.StatusForbidden, "forbidden")
			return
		}

		next(w, r)
	}
}

// UserIDFromContext ดึง user id ที่ RequireAuth ฝากไว้ใน context
// คืนค่า (id, true) ถ้ามี หรือ (0, false) ถ้าไม่มี (เช่นเผลอเรียกใน route ที่ไม่ได้ครอบ RequireAuth)
//
// ถูกเรียกจาก: handler เกือบทุกตัวในโฟลเดอร์ handlers/ เพื่อเอา user id ไปใส่ใน query
// เช่น WHERE user_id = ? ผู้ใช้แต่ละคนจึงเห็น/แก้ได้เฉพาะข้อมูลของตัวเอง
func UserIDFromContext(r *http.Request) (int, bool) {
	v, ok := r.Context().Value(userIDContextKey).(int)
	return v, ok
}

// UsernameFromContext ดึง username ที่ RequireAuth ฝากไว้ใน context (ทำงานแบบเดียวกับ UserIDFromContext)
// ตอนนี้ยังไม่มี handler ไหนเรียกใช้ เตรียมไว้เผื่อใช้ในอนาคต
func UsernameFromContext(r *http.Request) (string, bool) {
	v, ok := r.Context().Value(usernameContextKey).(string)
	return v, ok
}
