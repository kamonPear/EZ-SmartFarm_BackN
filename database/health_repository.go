// =====================================================================================
// ไฟล์ database/health_repository.go - ผลการตรวจสุขภาพไก่ (ตาราง health)
// =====================================================================================
//
// ผลตรวจ 1 ครั้ง = 1 แถว: คอก, จำนวนไก่สุขภาพดี, จำนวนไก่ป่วย, วันที่ตรวจ, หมายเหตุ
// ตารางมี unique key (coop_id, date) = คอกเดียวกันบันทึกผลตรวจซ้ำในวันเดียวกันไม่ได้
// ถูกเรียกจาก: handlers/health.go และ handlers/vaccine.go (แจ้งเตือนตรวจสุขภาพ)
package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// IsDuplicateHealthDate ตรวจว่า error จาก DB เกิดจาก "บันทึกผลตรวจซ้ำวันเดิม" หรือไม่
// (ข้อความ error มีทั้งคำว่า duplicate และ coop_id)
// handler จะได้ตอบว่า "มีผลตรวจของคอกนี้วันที่ ... อยู่แล้ว กรุณาแก้ไขข้อมูลเดิมแทน"
// ถูกเรียกจาก: handlers/health.go
func IsDuplicateHealthDate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") && strings.Contains(msg, "coop_id")
}

// normalizeHealthDate ตัดเวลาออกจากวันที่ เหลือแค่วัน (00:00 ตามเวลาท้องถิ่น)
// เพื่อให้ unique key (coop_id, date) กันซ้ำได้จริง ถ้ามีเวลาติดมา วันเดียวกันจะดูเหมือนคนละค่า
func normalizeHealthDate(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

// CreateHealth บันทึกผลตรวจสุขภาพใหม่ (ปุ่ม "บันทึกข้อมูล" หน้าตรวจสุขภาพไก่)
// เช็กก่อนว่าคอกเป็นของผู้ใช้ แล้วตัดเวลาออกจากวันที่ก่อนบันทึก
// ถ้าวันนั้นมีผลตรวจแล้ว DB จะคืน error ซ้ำ ซึ่ง handler แยกด้วย IsDuplicateHealthDate
// (ฝั่ง backend ตรวจแค่ว่าต้องมี coop_id และวันที่ ใน handlers/health.go)
// ถูกเรียกจาก: handlers/health.go -> CreateHealthHandler (POST /api/healths)
func CreateHealth(req *models.CreateHealthRequest, userID int) (*models.Health, error) {
	owned, err := CoopBelongsToUser(req.CoopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	normalizedDate := normalizeHealthDate(req.RecordDate)

	health := &models.Health{
		CoopID:     req.CoopID,
		Healthy:    req.Healthy,
		PoorHealth: req.PoorHealth,
		Note:       req.Note,
		RecordDate: normalizedDate,
	}

	if err := DB.Create(health).Error; err != nil {
		log.Printf("Error creating health record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Health record created with ID: %d for Coop: %d\n", health.HealthID, health.CoopID)
	return health, nil
}

// GetHealthByID ดึงผลตรวจ 1 รายการตาม id พร้อมข้อมูลคอก เฉพาะคอกของผู้ใช้ ไม่เจอคืน ErrNotFound
func GetHealthByID(id int, userID int) (*models.Health, error) {
	var h *models.Health

	if err := DB.Preload("Coop").
		Where("health_id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", id, userID).
		First(&h).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		log.Printf("Error fetching health record: %v", err)
		return nil, err
	}

	return h, nil
}

// GetHealthsByCoopID ดึงผลตรวจทั้งหมดของคอกเดียว เรียงจากวันล่าสุด (หลังเช็กว่าคอกเป็นของผู้ใช้)
// ใช้ในส่วน "ประวัติสุขภาพ" ของหน้าแก้ไขข้อมูลคอกไก่
func GetHealthsByCoopID(coopID int, userID int) ([]models.Health, error) {
	var hs []models.Health

	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	// 🌟 แก้ "record_date" -> "date" ให้ตรงกับชื่อคอลัมน์จริงในตาราง health
	if err := DB.Preload("Coop").Where("coop_id = ?", coopID).Order("date DESC").Find(&hs).Error; err != nil {
		log.Printf("Error fetching health records: %v", err)
		return nil, err
	}

	return hs, nil
}

// GetAllHealths ดึงผลตรวจของทุกคอกของผู้ใช้ เรียงจากวันล่าสุด ใช้ทำกราฟสรุปผลสุขภาพไก่
func GetAllHealths(userID int) ([]models.Health, error) {
	var hs []models.Health

	if err := DB.Preload("Coop").
		Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).
		Order("date DESC").Find(&hs).Error; err != nil {
		log.Printf("Error fetching all health records: %v", err)
		return nil, err
	}

	return hs, nil
}

// UpdateHealth แก้ไขผลตรวจ (หน้าต่าง "แก้ไข บันทึกสุขภาพ")
//   - เช็กว่าผลตรวจเป็นของผู้ใช้ ถ้าย้ายไปคอกอื่น คอกนั้นต้องเป็นของผู้ใช้ด้วย
//   - อัปเดตเฉพาะช่องที่ส่งมา โดยใช้ map ที่ key เป็น "ชื่อคอลัมน์จริงในตาราง"
//     (number_healthy, number_poor_health, date) ไม่ใช่ชื่อ field ใน Go
//     เพราะ GORM ไม่แปลงชื่อให้เมื่อส่งเป็น map
//   - จำนวนไก่ใช้ >= 0 เพราะ 0 ตัวเป็นค่าที่ถูกต้อง (เช่นไม่มีไก่ป่วยเลย)
func UpdateHealth(id int, req *models.UpdateHealthRequest, userID int) (*models.Health, error) {
	h, err := GetHealthByID(id, userID)
	if err != nil {
		return nil, err
	}

	// if coop change provided, validate it belongs to userID too
	if req.CoopID > 0 && req.CoopID != h.CoopID {
		owned, err := CoopBelongsToUser(req.CoopID, userID)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, ErrNotFound
		}
	}

	// 🌟 ตรงนี้ใช้ map[string]interface{} ส่งเข้า .Updates() โดยตรง
	// GORM จะมองว่า key ของ map คือ "ชื่อคอลัมน์จริง" ตรงๆ ไม่ได้แปลงผ่าน
	// gorm:"column:..." ใน struct เหมือนตอนใช้ struct (เช่นใน CreateHealth)
	// จึงต้องใช้ชื่อคอลัมน์จริง (number_healthy, number_poor_health, date)
	// ไม่ใช่ชื่อ field/JSON (healthy, poor_health, record_date)
	updates := map[string]interface{}{}
	if req.CoopID > 0 {
		updates["coop_id"] = req.CoopID
	}
	if req.Healthy >= 0 {
		updates["number_healthy"] = req.Healthy
	}
	if req.PoorHealth >= 0 {
		updates["number_poor_health"] = req.PoorHealth
	}
	if !req.RecordDate.IsZero() {
		updates["date"] = normalizeHealthDate(req.RecordDate)
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if len(updates) == 0 {
		return h, nil
	}

	if err := DB.Model(h).Updates(updates).Error; err != nil {
		log.Printf("Error updating health record: %v", err)
		return nil, err
	}

	fmt.Printf("✓ Health record %d updated\n", id)
	return h, nil
}

// DeleteHealth ลบผลตรวจ 1 รายการ (เช็กสิทธิ์ผ่าน GetHealthByID ก่อน)
func DeleteHealth(id int, userID int) error {
	h, err := GetHealthByID(id, userID)
	if err != nil {
		return err
	}

	if err := DB.Delete(h).Error; err != nil {
		log.Printf("Error deleting health record: %v", err)
		return err
	}

	fmt.Printf("✓ Health record %d deleted\n", id)
	return nil
}

// DeleteHealthsByCoopID ลบผลตรวจทั้งหมดของคอก "โดยไม่เช็กเจ้าของ" ตอนนี้ไม่มีที่ไหนเรียกใช้
func DeleteHealthsByCoopID(coopID int) error {
	if err := DB.Where("coop_id = ?", coopID).Delete(&models.Health{}).Error; err != nil {
		log.Printf("Error deleting health records for coop %d: %v", coopID, err)
		return err
	}

	fmt.Printf("✓ All health records for coop %d deleted\n", coopID)
	return nil
}

// ==========================================
// 🌟 ส่วนแจ้งเตือนตรวจสุขภาพ (Notifications)
// ==========================================

// HealthNotiDB โครงสร้างชั่วคราวใช้รับผลจาก SQL ดิบใน GetHealthCheckNotifications
// (is_today เป็น 1/0 เพราะ SQL คืนตัวเลข)
type HealthNotiDB struct {
	CoopID  int `gorm:"column:coop_id"`
	IsToday int `gorm:"column:is_today"`
}

// HealthNotification รูปแบบข้อมูลแจ้งเตือนที่ส่งกลับให้แอป: ชื่อคอก และวันนี้หรือพรุ่งนี้ (true/false)
type HealthNotification struct {
	CoopName string `json:"coop_name"`
	IsToday  bool   `json:"is_today"`
}

// GetHealthCheckNotifications ดึงคอกที่มีการตรวจสุขภาพ "วันนี้" หรือ "พรุ่งนี้" ของผู้ใช้
// ใช้ SQL ดิบ (DB.Raw) เพราะต้องใช้ CASE WHEN และ DATE() ซึ่งเขียนแบบ SQL ตรง ๆ อ่านง่ายกว่า
// แต่ยังส่งค่าผ่าน ? เสมอ จึงปลอดภัยจาก SQL injection
// JOIN coop เพื่อกรองเฉพาะคอกของผู้ใช้
// ข้อสังเกต: CoopName ใส่เป็น "เลข id ของคอก" (ไม่ใช่ชื่อคอกจริง) ตามรูปแบบที่แอปรุ่นเดิมใช้
// ถูกเรียกจาก: handlers/vaccine.go -> GetHealthCheckNotiHandler (GET /api/notifications/health_checks)
func GetHealthCheckNotifications(userID int) ([]HealthNotification, error) {
	var dbResults []HealthNotiDB

	now := time.Now().In(time.Local)
	todayStr := now.Format("2006-01-02")
	tomorrowStr := now.AddDate(0, 0, 1).Format("2006-01-02")

	// ดึง coop_id มาโดยตรงจากตาราง health เฉพาะคอกของ userID
	query := `
		SELECT health.coop_id,
		       CASE WHEN DATE(health.date) = ? THEN 1 ELSE 0 END as is_today
		FROM health
		JOIN coop ON coop.coop_id = health.coop_id
		WHERE (DATE(health.date) = ? OR DATE(health.date) = ?) AND coop.user_id = ?
	`

	err := DB.Raw(query, todayStr, todayStr, tomorrowStr, userID).Scan(&dbResults).Error
	if err != nil {
		log.Printf("Error fetching health check notifications: %v", err)
		return nil, err
	}

	// แปลงตัวเลข CoopID (เช่น 1) ให้เป็นข้อความ ("1") ส่งกลับให้แอป
	var notifications []HealthNotification
	for _, r := range dbResults {
		notifications = append(notifications, HealthNotification{
			CoopName: fmt.Sprintf("%d", r.CoopID),
			IsToday:  r.IsToday == 1,
		})
	}

	return notifications, nil
}
