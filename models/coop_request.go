// =====================================================================================
// ไฟล์ models/coop_request.go - รูปแบบ JSON ที่ส่งเข้ามาตอนเพิ่ม/แก้ไขคอก
// =====================================================================================
//
// หมายเหตุ tag binding:"required" ในไฟล์ *_request.go: เป็น tag ของเฟรมเวิร์ก Gin
// โปรเจกต์นี้ใช้ net/http ธรรมดา tag นี้จึงไม่มีผล การตรวจข้อมูลจริงเขียนไว้ใน handlers/
package models

import "time"

// CreateCoopRequest body ของ POST /api/coops (เพิ่มคอก)
// handlers/coop.go ตรวจว่ากรอก ชื่อ วันรับเข้า จำนวนไก่ วันเกิด ครบ และวันรับเข้าไม่ก่อนวันเกิด
type CreateCoopRequest struct {
	NameCoop         string    `json:"name_coop" binding:"required"`
	DateAdoptAnimals time.Time `json:"date_adopt_animals" binding:"required"`
	Amount           int       `json:"amount" binding:"required,min=1"`
	Birthday         time.Time `json:"birthday" binding:"required"`
	Note             string    `json:"note"`
}

// UpdateCoopRequest body ของ PUT /api/coops?id= (แก้ไขคอก)
// ช่องที่ส่งค่าว่าง/ศูนย์ จะไม่ถูกอัปเดต (database.UpdateCoop)
type UpdateCoopRequest struct {
	NameCoop         string    `json:"name_coop"`
	DateAdoptAnimals time.Time `json:"date_adopt_animals"`
	Amount           int       `json:"amount" binding:"min=1"`
	Birthday         time.Time `json:"birthday"`
	Note             string    `json:"note"`
}

// CoopResponse รูปแบบตอบกลับข้อมูลคอกแบบย่อ ตอนนี้ไม่มีที่ไหนใช้ (API ส่ง Coop ตรง ๆ)
type CoopResponse struct {
	CoopID           int       `json:"coop_id"`
	DateAdoptAnimals time.Time `json:"date_adopt_animals"`
	Amount           int       `json:"amount"`
	Birthday         time.Time `json:"birthday"`
	Note             string    `json:"note"`
}
