package models

import "time"

// โครงสร้างสำหรับตาราง User ในฐานข้อมูล
type User struct {
	// ใช้ tag gorm เพื่อแมพให้ตรงกับฐานข้อมูลที่คุณสร้างใน DBeaver
	// type:int ต้องระบุให้ชัด - ไม่งั้น GORM มองคอลัมน์นี้เป็น bigint ตามดีฟอลต์ของ
	// Go int แล้วไล่ขยาย user_id ในตารางลูกให้เป็น bigint ตาม ซึ่ง FK บล็อกไว้
	// ทำให้ AutoMigrate ล้มตั้งแต่ model แรกแล้วข้าม model ที่เหลือทั้งหมด
	ID        int       `json:"id" gorm:"column:id_User;primaryKey;autoIncrement;type:int;size:32"`
	Username  string    `json:"username" gorm:"column:username;unique;not null"`
	Password  string    `json:"-" gorm:"column:password;not null"`
	Role      string    `json:"role" gorm:"column:role;type:varchar(20);not null;default:'user'"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// บังคับให้ GORM รู้ว่าตารางนี้ใน Database ชื่อ "User" (ตรงกับที่คุณสร้าง)
func (User) TableName() string {
	return "User"
}

// โครงสร้างสำหรับรับข้อมูลตอน Login (เหมือนเดิม)
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest represents the request payload for creating a new user (admin-only).
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // optional, defaults to "user"
}
