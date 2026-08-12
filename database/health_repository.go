package database

import (
	"fmt"
	"log"
	"time"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

func normalizeHealthDate(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

// CreateHealth creates a new health record in the database
func CreateHealth(req *models.CreateHealthRequest) (*models.Health, error) {
	// Verify coop exists
	if _, err := GetCoopByID(req.CoopID); err != nil {
		return nil, fmt.Errorf("coop not found: %v", err)
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

// GetHealthByID retrieves a health record by ID
func GetHealthByID(id int) (*models.Health, error) {
	var h *models.Health

	if err := DB.Preload("Coop").Where("health_id = ?", id).First(&h).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("health record not found")
		}
		log.Printf("Error fetching health record: %v", err)
		return nil, err
	}

	return h, nil
}

// GetHealthsByCoopID retrieves all health records for a specific coop
func GetHealthsByCoopID(coopID int) ([]models.Health, error) {
	var hs []models.Health

	// Verify coop exists
	if _, err := GetCoopByID(coopID); err != nil {
		return nil, fmt.Errorf("coop not found: %v", err)
	}

	// 🌟 แก้ "record_date" -> "date" ให้ตรงกับชื่อคอลัมน์จริงในตาราง health
	if err := DB.Preload("Coop").Where("coop_id = ?", coopID).Order("date DESC").Find(&hs).Error; err != nil {
		log.Printf("Error fetching health records: %v", err)
		return nil, err
	}

	return hs, nil
}

// GetAllHealths retrieves all health records
func GetAllHealths() ([]models.Health, error) {
	var hs []models.Health

	// 🌟 แก้ "record_date" -> "date" ให้ตรงกับชื่อคอลัมน์จริงในตาราง health
	if err := DB.Preload("Coop").Order("date DESC").Find(&hs).Error; err != nil {
		log.Printf("Error fetching all health records: %v", err)
		return nil, err
	}

	return hs, nil
}

// UpdateHealth updates an existing health record
func UpdateHealth(id int, req *models.UpdateHealthRequest) (*models.Health, error) {
	h, err := GetHealthByID(id)
	if err != nil {
		return nil, err
	}

	// if coop change provided, validate
	if req.CoopID > 0 && req.CoopID != h.CoopID {
		if _, err := GetCoopByID(req.CoopID); err != nil {
			return nil, fmt.Errorf("target coop not found: %v", err)
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

// DeleteHealth deletes a health record
func DeleteHealth(id int) error {
	h, err := GetHealthByID(id)
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

// DeleteHealthsByCoopID deletes all health records for a coop
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

// HealthNotiDB โครงสร้างชั่วคราวเพื่อรับค่าจาก Database (ใช้ตัวเลข CoopID)
type HealthNotiDB struct {
	CoopID  int `gorm:"column:coop_id"`
	IsToday int `gorm:"column:is_today"`
}

// HealthNotification โครงสร้างข้อมูลสำหรับแปลงตัวเลขเป็นข้อความก่อนส่งไปให้แอป
type HealthNotification struct {
	CoopName string `json:"coop_name"`
	IsToday  bool   `json:"is_today"`
}

// GetHealthCheckNotifications ดึงข้อมูลคอกที่ต้องตรวจสุขภาพของ "วันนี้" และ "พรุ่งนี้"
func GetHealthCheckNotifications() ([]HealthNotification, error) {
	var dbResults []HealthNotiDB

	now := time.Now().In(time.Local)
	todayStr := now.Format("2006-01-02")
	tomorrowStr := now.AddDate(0, 0, 1).Format("2006-01-02")

	// ดึง coop_id มาโดยตรงจากตาราง health
	query := `
		SELECT coop_id, 
		       CASE WHEN DATE(date) = ? THEN 1 ELSE 0 END as is_today 
		FROM health
		WHERE DATE(date) = ? OR DATE(date) = ?
	`
	
	err := DB.Raw(query, todayStr, todayStr, tomorrowStr).Scan(&dbResults).Error
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