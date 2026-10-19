// =====================================================================================
// ไฟล์ database/health_appointment_repository.go - "นัดตรวจสุขภาพเอง" (นัดล่วงหน้าแบบกำหนดวันเอง)
// =====================================================================================
//
// ระบบนัดตรวจสุขภาพมี 2 โหมด (หน้า "นัดตรวจสุขภาพ"):
//   - นัดให้อัตโนมัติ : คำนวณจากวันให้วัคซีน (ตรวจก่อนวันให้วัคซีน 1 วัน) อยู่ใน handlers/vaccine.go
//   - นัดตรวจสุขภาพเอง: ผู้ใช้เลือกวันเอง เก็บในตาราง health_appointments <- ไฟล์นี้
//
// เดิมโหมดนัดเองเก็บใน localStorage ของเว็บ ย้ายมาเก็บใน DB เพื่อให้แอปเห็นด้วย
// ถูกเรียกจาก: handlers/health_appointment.go
package database

import (
	"errors"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// normalizeAppointmentDate ตัดเวลาออก เหลือแค่วันที่ (เที่ยงคืน 00:00 ตามเวลาท้องถิ่น)
// ทำไมต้องทำ: วันที่ที่ส่งมาอาจมีเวลาติดมาด้วย (เช่น 2026-10-12 14:30) ถ้าไม่ตัด
// นัดวันเดียวกันจะถูกมองว่าเป็นคนละนัด และเช็กนัดซ้ำไม่เจอ
func normalizeAppointmentDate(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

// CreateHealthAppointment สร้างนัดตรวจสุขภาพให้คอก (ปุ่ม "+ เพิ่มนัดตรวจใหม่")
//
// ขั้นตอน:
//  1. CoopBelongsToUser ตรวจว่าคอกนี้เป็นของผู้ใช้จริง ถ้าไม่ใช่ คืน ErrNotFound
//     (กันการสร้างนัดให้คอกของฟาร์มอื่น โดยตอบว่า "ไม่พบ" ไม่บอกว่ามีคอกนี้อยู่)
//  2. ตัดเวลาออกจากวันที่นัด
//  3. ถ้าคอกนี้มีนัดวันนี้อยู่แล้ว -> คืนนัดเดิม (ไม่ error ไม่สร้างซ้ำ) เรียกว่า idempotent
//     กดซ้ำกี่ครั้งผลก็เหมือนเดิม
//  4. ยังไม่มี -> บันทึกนัดใหม่
func CreateHealthAppointment(req *models.CreateHealthAppointmentRequest, userID int) (*models.HealthAppointment, error) {
	owned, err := CoopBelongsToUser(req.CoopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	date := normalizeAppointmentDate(req.AppointmentDate)

	var existing models.HealthAppointment
	if err := DB.Where("coop_id = ? AND appointment_date = ?", req.CoopID, date).First(&existing).Error; err == nil {
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Error checking existing health appointment: %v", err)
		return nil, err
	}

	appt := &models.HealthAppointment{
		CoopID:          req.CoopID,
		AppointmentDate: date,
	}
	if err := DB.Create(appt).Error; err != nil {
		log.Printf("Error creating health appointment: %v", err)
		return nil, err
	}

	log.Printf("✓ Health appointment created with ID: %d for Coop: %d on %s", appt.AppointmentID, appt.CoopID, date.Format("2006-01-02"))
	return appt, nil
}

// GetHealthAppointments ดึงนัดตรวจ "ตั้งแต่วันนี้เป็นต้นไป" ของทุกคอกที่ผู้ใช้เป็นเจ้าของ เรียงจากใกล้สุด
// ถ้าส่ง coopID มา จะกรองเฉพาะคอกนั้น (ใช้ pointer *int เพื่อแยก "ไม่ส่งมา" (nil) ออกจาก "ส่ง 0")
//
// coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?) = subquery กรองเฉพาะคอกของผู้ใช้
// นัดที่ผ่านไปแล้วยังเก็บไว้ในตาราง แต่ไม่ดึงมาแสดง เพราะพอถึงวันก็กลายเป็นการตรวจสุขภาพปกติแล้ว
func GetHealthAppointments(userID int, coopID *int) ([]models.HealthAppointment, error) {
	var appts []models.HealthAppointment
	today := normalizeAppointmentDate(time.Now())

	q := DB.Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Where("appointment_date >= ?", today)
	if coopID != nil {
		q = q.Where("coop_id = ?", *coopID)
	}
	if err := q.Order("appointment_date ASC").Find(&appts).Error; err != nil {
		log.Printf("Error fetching health appointments: %v", err)
		return nil, err
	}
	return appts, nil
}

// DeleteHealthAppointment ลบนัดตรวจ โดยต้องเป็นนัดของคอกที่ผู้ใช้เป็นเจ้าของเท่านั้น
// หาไม่เจอ (หรือเป็นของคนอื่น) -> ErrNotFound ซึ่ง handler จะตอบ 404
func DeleteHealthAppointment(id int, userID int) error {
	var appt models.HealthAppointment
	err := DB.Where("appointment_id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", id, userID).
		First(&appt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		log.Printf("Error fetching health appointment to delete: %v", err)
		return err
	}

	if err := DB.Delete(&appt).Error; err != nil {
		log.Printf("Error deleting health appointment: %v", err)
		return err
	}

	log.Printf("✓ Health appointment %d deleted", id)
	return nil
}
