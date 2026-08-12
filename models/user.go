package models

// โครงสร้างสำหรับตาราง User ในฐานข้อมูล
type User struct {
	// ใช้ tag gorm เพื่อแมพให้ตรงกับฐานข้อมูลที่คุณสร้างใน DBeaver
	ID       int    `json:"id" gorm:"column:id_User;primaryKey;autoIncrement"`
	Username string `json:"username" gorm:"column:username;unique;not null"`
	Password string `json:"password" gorm:"column:password;not null"`
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