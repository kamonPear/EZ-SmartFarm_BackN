// =====================================================================================
// ไฟล์ models/user.go - โครงสร้างตาราง User (บัญชีผู้ใช้) และ JSON ตอนเข้าสู่ระบบ/สร้างบัญชี
// =====================================================================================
package models

import "time"

// User บัญชีผู้ใช้ 1 คน = เจ้าของฟาร์ม 1 ฟาร์ม (ตาราง User)
//  - ID       : คอลัมน์ id_User ระบุ type:int ชัดเจน (ถ้าไม่ระบุ GORM ใช้ bigint ตาม int ของ Go
//               แล้ว user_id ในตารางลูกต้องเป็น bigint ตาม Foreign Key ไม่ยอม ทำให้ AutoMigrate ล้มทั้งหมด)
//  - Username : ห้ามซ้ำ (unique)
//  - Password : เก็บเป็น bcrypt hash ไม่ใช่รหัสจริง และ json:"-" = ไม่ส่งออกทาง API เด็ดขาด
type User struct {
	// ใช้ tag gorm เพื่อแมพให้ตรงกับฐานข้อมูลที่คุณสร้างใน DBeaver
	// type:int ต้องระบุให้ชัด - ไม่งั้น GORM มองคอลัมน์นี้เป็น bigint ตามดีฟอลต์ของ
	// Go int แล้วไล่ขยาย user_id ในตารางลูกให้เป็น bigint ตาม ซึ่ง FK บล็อกไว้
	// ทำให้ AutoMigrate ล้มตั้งแต่ model แรกแล้วข้าม model ที่เหลือทั้งหมด
	ID        int       `json:"id" gorm:"column:id_User;primaryKey;autoIncrement;type:int;size:32"`
	Username  string    `json:"username" gorm:"column:username;unique;not null"`
	Password  string    `json:"-" gorm:"column:password;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName บอก GORM ว่าใช้ตารางชื่อ "User" (ตัว U ใหญ่ ตรงกับตารางที่สร้างไว้เดิมในฐานข้อมูล)
func (User) TableName() string {
	return "User"
}

// LoginRequest body ของ POST /api/auth/login: ชื่อผู้ใช้ และรหัสผ่าน
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest body ของ POST /api/auth/register (ต้องมี header X-Admin-Key ไม่ได้ใช้ role ของผู้ใช้)
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
