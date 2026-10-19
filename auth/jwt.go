// =====================================================================================
// ไฟล์ auth/jwt.go - ระบบ "บัตรผ่าน" (JWT Token) ของการเข้าสู่ระบบ
// =====================================================================================
//
// ภาพรวมการทำงาน (อ่านอันนี้ก่อน):
//
//  1. ผู้ใช้กรอก username/password ที่หน้า Login (เว็บ Angular หรือแอป Flutter)
//     -> ส่งมาที่ POST /api/auth/login (routes/routes.go บรรทัด ~34)
//     -> handlers/auth.go: LoginHandler ตรวจรหัสผ่านถูกแล้ว จะเรียก GenerateToken() ในไฟล์นี้
//     -> ได้ token กลับไปให้ฝั่งหน้าบ้าน
//  2. หน้าบ้านเก็บ token ไว้
//     - เว็บ: เก็บใน localStorage key "ez_auth_session" (services/auth.service.ts)
//     - แอป: เก็บใน SharedPreferences
//  3. ทุกครั้งที่หน้าบ้านเรียก API อื่น (ดึงคอก, บันทึกไข่, ฯลฯ) จะแนบ header
//     Authorization: Bearer <token>
//  4. ฝั่ง backend มี auth.RequireAuth (auth/middleware.go) ครอบ route ไว้ประมาณ 59 route
//     -> RequireAuth เรียก ParseToken() ในไฟล์นี้ เพื่อตรวจว่า token แท้และยังไม่หมดอายุ
//     -> ถ้าผ่าน ดึง user id ด้วย claims.UserID() แล้วฝากไว้ใน context
//        handler แต่ละตัวจึงรู้ว่า "ใครเป็นคนเรียก" และดึงเฉพาะข้อมูลฟาร์มของคนนั้น
//
// ทำไมต้องใช้ JWT (JSON Web Token)?
//   - เซิร์ฟเวอร์ไม่ต้องเก็บ session ไว้ในฐานข้อมูล ข้อมูลผู้ใช้ (id, username, วันหมดอายุ)
//     อยู่ใน token เลย แล้วมี "ลายเซ็น" กำกับไว้ ใครแก้ข้อมูลใน token ลายเซ็นจะไม่ตรงทันที
//   - ใช้ได้ทั้งเว็บและแอปด้วยวิธีเดียวกัน (แค่แนบ header) เหมาะกับระบบที่มี 2 หน้าบ้าน
//   - backend อยู่บน Render ซึ่งอาจรีสตาร์ทได้ตลอด ถ้าเก็บ session ไว้ในหน่วยความจำ
//     ผู้ใช้จะหลุดทุกครั้งที่รีสตาร์ท แต่ JWT ไม่หลุด เพราะตรวจได้จากลายเซ็นอย่างเดียว
//
// หน้าตาของ JWT: xxxxx.yyyyy.zzzzz (3 ส่วนคั่นด้วยจุด)
//   - xxxxx = header   บอกว่าเซ็นด้วยวิธีไหน (ที่นี่คือ HS256)
//   - yyyyy = payload  ข้อมูลผู้ใช้ (ดู struct Claims ด้านล่าง) - อ่านได้แต่แก้ไม่ได้
//   - zzzzz = signature ลายเซ็นที่สร้างจาก header+payload+รหัสลับ (JWT_SECRET)
//   หมายเหตุ: payload แค่ถูกเข้ารหัสแบบ base64 ไม่ได้ถูกซ่อน จึงห้ามใส่รหัสผ่านลงใน token
// =====================================================================================

package auth

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	// ไลบรารีมาตรฐานของ Go สำหรับสร้าง/ตรวจ JWT (ติดตั้งผ่าน go.mod)
	// เลือกใช้ v5 เพราะเป็นเวอร์ชันล่าสุดที่ยังดูแลอยู่ และบังคับตรวจวันหมดอายุให้อัตโนมัติ
	"github.com/golang-jwt/jwt/v5"
)

// TokenTTL = อายุของ token หลังออกให้ = 168 ชั่วโมง = 7 วัน
// (TTL ย่อมาจาก Time To Live) เมื่อเกิน 7 วัน ParseToken จะปฏิเสธ token นี้
// และผู้ใช้ต้อง login ใหม่
//
// ข้อสังเกต: ฝั่งเว็บตั้งให้จำการ login แค่ 24 ชม. (SESSION_TTL_MS ในเว็บ) ซึ่งสั้นกว่า
// เว็บจึงบังคับ login ใหม่ก่อน token หมดอายุจริง ส่วนแอปจะใช้ได้ถึง 7 วัน
const TokenTTL = 168 * time.Hour

// devJWTSecret = รหัสลับสำรอง ใช้ "เฉพาะตอนพัฒนาในเครื่องตัวเอง" เท่านั้น
//
// ปกติรหัสลับที่ใช้เซ็น token ต้องมาจาก environment variable ชื่อ JWT_SECRET
// (ตั้งในไฟล์ .env ตอนรันในเครื่อง หรือหน้า Environment ของ Render ตอนใช้งานจริง)
// ที่ต้องมีตัวสำรองนี้ไว้ เพื่อให้คนในทีมรันโปรเจกต์ได้ทันทีแม้ยังไม่มีไฟล์ .env
//
// ห้ามใช้ตัวนี้ตอนใช้งานจริง เพราะใครเปิดอ่านโค้ดนี้ (เช่นบน GitHub) ก็รู้รหัสลับ
// แล้วปลอม token เป็นผู้ใช้คนไหนก็ได้
const devJWTSecret = "dev-only-insecure-jwt-secret-do-not-use-in-production-3f8a2c91"

var (
	// secretOnce ประกาศไว้แต่ตอนนี้ไม่มีโค้ดส่วนไหนเรียกใช้ (เหลือจากโค้ดเดิม ลบทิ้งได้)
	secretOnce sync.Once

	// secretWarn ใช้คู่กับ secretWarn.Do(...) ในฟังก์ชัน jwtSecret()
	// sync.Once ทำให้โค้ดข้างใน Do ทำงาน "ครั้งเดียวตลอดอายุโปรแกรม"
	// ข้อความเตือนจึงขึ้นแค่ครั้งเดียว ไม่ขึ้นซ้ำทุกครั้งที่มีคน login
	secretWarn sync.Once
)

// jwtSecret คืนค่า "รหัสลับ" ที่ใช้เซ็นและตรวจลายเซ็นของ token
//
// ทำงานอย่างไร:
//   - อ่าน JWT_SECRET จาก environment ด้วย os.Getenv
//   - ถ้ามีค่า -> ใช้ค่านั้น (กรณีใช้งานจริงบน Render)
//   - ถ้าไม่มี -> พิมพ์คำเตือนลง log ครั้งเดียว แล้วใช้ devJWTSecret แทน
//
// ถูกเรียกจากไหน: GenerateToken() (ตอนเซ็น) และ ParseToken() (ตอนตรวจ) ในไฟล์นี้
//
// ทำไมคืนเป็น []byte: เพราะวิธีเซ็นแบบ HS256 ของไลบรารี jwt ต้องการกุญแจเป็น byte
//
// ทำไมต้องใช้กุญแจตัวเดียวกันทั้งตอนเซ็นและตอนตรวจ: HS256 เป็นการเข้ารหัสแบบ
// "กุญแจเดียว" (symmetric) ถ้ารหัสลับเปลี่ยน token เก่าทั้งหมดจะใช้ไม่ได้ทันที
// (ผู้ใช้ทุกคนต้อง login ใหม่) ซึ่งเป็นวิธีบังคับ logout ทุกคนได้ด้วย
func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	secretWarn.Do(func() {
		log.Println("=====================================================================")
		log.Println("WARNING: JWT_SECRET is not set in the environment. Falling back to a")
		log.Println("fixed, publicly-known development secret. This is INSECURE and must")
		log.Println("NEVER be used in a real deployment - set JWT_SECRET in .env / the")
		log.Println("environment before deploying.")
		log.Println("=====================================================================")
	})
	return []byte(devJWTSecret)
}

// Claims = "ข้อมูลที่ฝังอยู่ใน token" (ส่วน payload ของ JWT)
//
// ประกอบด้วย 2 ส่วน:
//   - Username : ชื่อผู้ใช้ (field ที่เราเพิ่มเอง) เก็บไว้ให้ middleware ฝากต่อใน context
//     ได้เลยโดยไม่ต้องค้นฐานข้อมูลอีกรอบ
//   - jwt.RegisteredClaims : field มาตรฐานของ JWT ที่ไลบรารีเตรียมไว้ให้ ที่ใช้จริงมี
//   - Subject   (sub) = user id ของผู้ใช้ (เก็บเป็นข้อความ เพราะมาตรฐานกำหนดให้เป็น string)
//   - IssuedAt  (iat) = เวลาที่ออก token
//   - ExpiresAt (exp) = เวลาหมดอายุ ไลบรารีจะตรวจให้อัตโนมัติตอน ParseToken
//
// การเขียน jwt.RegisteredClaims ไว้ใน struct โดยไม่ตั้งชื่อ field เรียกว่า "embedding"
// ทำให้เรียก c.Subject, c.ExpiresAt ได้ตรง ๆ เหมือนเป็น field ของ Claims เอง
//
// แท็ก `json:"username"` กำหนดชื่อ key ใน JSON ของ token เป็น "username"
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken สร้าง token ใหม่ให้ผู้ใช้ที่ login สำเร็จ
//
// รับค่า:  userID (เลข id ผู้ใช้จากตาราง users), username (ชื่อผู้ใช้)
// คืนค่า:  token เป็นข้อความยาว ๆ แบบ xxxxx.yyyyy.zzzzz หรือ error ถ้าเซ็นไม่สำเร็จ
//
// ถูกเรียกจากไหน: handlers/auth.go -> LoginHandler หลังตรวจว่า user มีอยู่จริงและ
// รหัสผ่านถูกต้อง (auth.CheckPassword) แล้ว token จะถูกส่งกลับใน JSON
// รูปแบบ {"token": "...", "user": {"id": ..., "username": "..."}}
//
// ขั้นตอน:
//  1. จดเวลาปัจจุบันไว้ใน now (ใช้ค่าเดียวกันทั้ง IssuedAt และ ExpiresAt จะได้ตรงกันเป๊ะ)
//  2. สร้าง Claims ใส่ username, user id (แปลง int -> string ด้วย strconv.Itoa),
//     เวลาออก และเวลาหมดอายุ (= now + 7 วัน)
//  3. jwt.NewWithClaims(...) ประกอบ token โดยระบุวิธีเซ็นเป็น HS256
//  4. SignedString(jwtSecret()) เซ็นด้วยรหัสลับ แล้วแปลงเป็นข้อความพร้อมส่ง
//
// ทำไมเลือก HS256 (HMAC-SHA256):
//   - backend เป็นทั้งผู้ออกและผู้ตรวจ token เอง จึงใช้กุญแจลับตัวเดียวได้ ไม่ต้องมีกุญแจคู่
//     แบบ RS256 (ที่เหมาะกับกรณีให้ระบบอื่นตรวจ token ได้ด้วย)
//   - เร็ว ง่าย และเป็นค่ามาตรฐานที่ไลบรารี JWT ทุกภาษารองรับ
func GenerateToken(userID int, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ParseToken ตรวจ token ที่หน้าบ้านแนบมา ว่า "ของแท้" และ "ยังไม่หมดอายุ"
// ถ้าผ่านจะคืนข้อมูลข้างใน (Claims) กลับไป
//
// รับค่า:  tokenStr = ข้อความ token (ส่วนหลังคำว่า "Bearer " ใน header Authorization)
// คืนค่า:  *Claims (มี username และ user id) หรือ error ถ้า token ใช้ไม่ได้
//
// ถูกเรียกจากไหน: auth/middleware.go -> RequireAuth ซึ่งครอบ route เกือบทุกตัวใน
// routes/routes.go เช่น /api/coops, /api/eggs ถ้า ParseToken คืน error
// RequireAuth จะตอบ 401 {"error":"unauthorized"} ทันที หน้าบ้านเห็น 401 แล้ว
// จะพาผู้ใช้กลับไปหน้า Login
//
// ขั้นตอน:
//  1. เตรียม Claims เปล่าไว้รับข้อมูล
//  2. jwt.ParseWithClaims(...) จะ
//     - แยก token เป็น 3 ส่วน แล้วแปลง payload ใส่ใน claims
//     - เรียกฟังก์ชันที่เราส่งให้ (func(t *jwt.Token) ...) เพื่อขอรหัสลับไว้ตรวจลายเซ็น
//     - คำนวณลายเซ็นใหม่ด้วยรหัสลับ แล้วเทียบกับลายเซ็นใน token ถ้าไม่ตรง แปลว่า token ถูกแก้หรือปลอม
//     - ตรวจ ExpiresAt ว่าเลยเวลาแล้วหรือยัง
//  3. jwt.WithValidMethods([]string{"HS256"}) = รับเฉพาะ token ที่เซ็นด้วย HS256 เท่านั้น
//     จุดนี้สำคัญด้านความปลอดภัย: กันการโจมตีแบบ "alg confusion" ที่คนร้ายแก้ header
//     เป็น alg "none" (ไม่มีลายเซ็น) หรือวิธีอื่น เพื่อหลอกให้ระบบยอมรับ token ปลอม
//  4. เช็ก token.Valid ซ้ำอีกชั้นเผื่อไว้ แล้วคืน claims
//
// กรณีที่จะได้ error: ไม่ใช่รูปแบบ JWT, ลายเซ็นไม่ตรง (ถูกแก้/ใช้รหัสลับอื่น),
// หมดอายุแล้ว (เกิน 7 วัน), หรือใช้วิธีเซ็นที่ไม่ใช่ HS256
func ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	return claims, nil
}

// UserID แปลง Subject (user id ที่เก็บเป็นข้อความใน token) กลับเป็นตัวเลข int
//
// ทำไมต้องแปลง: มาตรฐาน JWT กำหนดให้ Subject เป็น string ตอน GenerateToken จึงแปลง
// int -> string (strconv.Itoa) ตอนนี้ต้องแปลงกลับด้วย strconv.Atoi เพราะฐานข้อมูล
// ใช้ user_id เป็นตัวเลข
//
// ถูกเรียกจากไหน: auth/middleware.go -> RequireAuth หลัง ParseToken ผ่าน
// ได้ userID แล้วจะฝากไว้ใน context ของ request จากนั้น handler ต่าง ๆ เรียก
// auth.UserIDFromContext(r) เพื่อเอาไปกรองข้อมูล เช่น
// SELECT ... FROM coop WHERE user_id = ? ผู้ใช้แต่ละคนจึงเห็นเฉพาะฟาร์มของตัวเอง
//
// (c *Claims) หน้าชื่อฟังก์ชัน แปลว่าเป็น "method" ของ Claims เรียกใช้แบบ claims.UserID()
func (c *Claims) UserID() (int, error) {
	return strconv.Atoi(c.Subject)
}
