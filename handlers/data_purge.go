package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
	"EZ-SmartFarm_BachN/services"

	"gorm.io/gorm"
)

type purgeRequest struct {
	Categories []string `json:"categories"`
}

// ownedCoops คือเงื่อนไข "คอกของผู้ใช้คนที่เรียก" ที่ใช้กรองทุกตารางลูก
const ownedCoops = "coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)"

// purgeableCategories คือหมวดข้อมูลที่ลบทิ้งได้หลังสำรองข้อมูล (เฉพาะ "รายการบันทึก" ของผู้เรียกเอง) -
// ตั้งใจไม่รวมข้อมูลหลักของระบบ: ตัวคอก (ลบแล้วข้อมูลลูกหายหมด), ประเภทวัคซีน/ยา, ค่ามาตรฐานฟาร์ม และ
// คลังอาหาร (ประวัตินำออกถูกใช้ตัดสต็อกอัตโนมัติรายวัน ลบแล้วอาจตัดสต็อกซ้ำ)
var purgeableCategories = map[string]func(tx *gorm.DB, userID int) (int64, error){
	"eggs": func(tx *gorm.DB, userID int) (int64, error) {
		res := tx.Where(ownedCoops, userID).Delete(&models.Egg{})
		return res.RowsAffected, res.Error
	},
	"health": func(tx *gorm.DB, userID int) (int64, error) {
		res := tx.Where(ownedCoops, userID).Delete(&models.Health{})
		return res.RowsAffected, res.Error
	},
	"vaccine_history": func(tx *gorm.DB, userID int) (int64, error) {
		res := tx.Where(ownedCoops, userID).Delete(&models.VaccineHistory{})
		return res.RowsAffected, res.Error
	},
	"appointments": func(tx *gorm.DB, userID int) (int64, error) {
		res := tx.Where(ownedCoops, userID).Delete(&models.HealthAppointment{})
		return res.RowsAffected, res.Error
	},
	"devices": func(tx *gorm.DB, userID int) (int64, error) {
		// ลบประวัติค่าเซนเซอร์ของอุปกรณ์เหล่านี้ก่อน ไม่ให้เหลือข้อมูลกำพร้า
		if err := tx.Exec(
			"DELETE FROM sensor_log WHERE device_id IN (SELECT device_id FROM device WHERE "+ownedCoops+")", userID,
		).Error; err != nil {
			return 0, err
		}
		res := tx.Where(ownedCoops, userID).Delete(&models.Device{})
		return res.RowsAffected, res.Error
	},
}

// PurgeDataHandler ลบข้อมูลตามหมวดที่ระบุ (ใช้หลังผู้ใช้สำรองข้อมูลแล้วเลือก "ลบข้อมูลนี้") - ลบเฉพาะ
// ข้อมูลของผู้เรียกเอง และเฉพาะหมวดใน purgeableCategories ทำในทรานแซกชันเดียว (สำเร็จทั้งหมดหรือไม่ลบเลย)
// POST /api/data/purge  {"categories": ["eggs", "health", ...]}
// ตอบ {"deleted": {"eggs": 60, ...}}
func PurgeDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req purgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Categories) == 0 {
		writeJSONErr(w, http.StatusBadRequest, "categories is required")
		return
	}
	for _, c := range req.Categories {
		if _, known := purgeableCategories[c]; !known {
			writeJSONErr(w, http.StatusBadRequest, "category not allowed: "+c)
			return
		}
	}

	deleted := make(map[string]int64, len(req.Categories))
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, c := range req.Categories {
			if _, done := deleted[c]; done {
				continue
			}
			n, err := purgeableCategories[c](tx, userID)
			if err != nil {
				return err
			}
			deleted[c] = n
		}
		return nil
	})
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to purge data: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		writeJSONErr(w, http.StatusInternalServerError, "Failed to delete data")
		return
	}

	// ลบอุปกรณ์แล้ว แจ้งบอร์ดของทุกคอกว่าไม่มีอุปกรณ์เหลือ (ส่งรายการเลขช่องว่าง)
	if _, purgedDevices := deleted["devices"]; purgedDevices {
		var coopIDs []int32
		if err := database.DB.Model(&models.Coop{}).Where("user_id = ?", userID).Pluck("coop_id", &coopIDs).Error; err == nil {
			go func() {
				for _, id := range coopIDs {
					services.PublishSlotsForCoop(id)
				}
			}()
		}
	}

	log.Printf("[%s] %s - %d ✓ Purged data for user %d: %v", r.Method, r.RequestURI, http.StatusOK, userID, deleted)
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": deleted})
}
