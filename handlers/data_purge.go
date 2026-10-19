// =====================================================================================
// ไฟล์ handlers/data_purge.go - API ลบข้อมูลบันทึกทีละหมวด (หลังผู้ใช้สำรองข้อมูลแล้ว)
// =====================================================================================
//
// หน้าเว็บมีส่วน "สำรองข้อมูล" ให้ดาวน์โหลดข้อมูลเก็บไว้ แล้วเลือก "ลบข้อมูลนี้" เพื่อล้างของเก่า
// ไฟล์นี้ลบได้เฉพาะหมวดที่อนุญาต และเฉพาะข้อมูลของผู้ที่ล็อกอินเท่านั้น
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

// purgeRequest รูปแบบ body ที่หน้าเว็บส่งมา เช่น {"categories": ["eggs", "health"]}
type purgeRequest struct {
	Categories []string `json:"categories"`
}

// ownedCoops เงื่อนไข SQL "เฉพาะคอกของผู้ใช้คนนี้" ใช้กรองทุกตารางลูก (ไข่ สุขภาพ วัคซีน ฯลฯ)
// เขียนเป็นค่าคงที่ไว้ที่เดียว ใช้ซ้ำได้และลดโอกาสพิมพ์ผิดจนลบข้อมูลคนอื่น
const ownedCoops = "coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)"

// purgeableCategories รายการหมวดที่ลบได้ และวิธีลบของแต่ละหมวด (map ชื่อหมวด -> ฟังก์ชันลบ)
//   - eggs            : บันทึกการเก็บไข่
//   - health          : ผลตรวจสุขภาพ
//   - vaccine_history : ประวัติการให้วัคซีน
//   - appointments    : นัดตรวจสุขภาพ
//   - devices         : อุปกรณ์ในคอก (ลบค่าเซนเซอร์ใน sensor_log ก่อน ไม่ให้เหลือข้อมูลกำพร้า)
//
// ตั้งใจไม่ให้ลบ: ตัวคอก (ลบแล้วข้อมูลลูกหายหมด), ประเภทวัคซีน/ยา, ค่ามาตรฐาน, คลังอาหาร
// (ประวัติอาหารถูกใช้กันการตัดสต็อกซ้ำ ลบแล้วอาจตัดซ้ำได้)
// ทำไมใช้ map ของฟังก์ชัน: หมวดที่ไม่อยู่ใน map ลบไม่ได้แน่นอน (whitelist) และเพิ่มหมวดใหม่ได้ง่าย
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

// PurgeDataHandler (POST /api/data/purge) ลบข้อมูลตามหมวดที่เลือก
// body: {"categories": ["eggs", "health", ...]} ตอบ {"deleted": {"eggs": 60, ...}} (จำนวนแถวที่ลบ)
//
// ขั้นตอน:
//  1. ตรวจว่าส่งหมวดมาอย่างน้อย 1 หมวด และทุกหมวดอยู่ใน purgeableCategories ไม่งั้น 400
//  2. ลบทุกหมวดใน Transaction เดียว = สำเร็จทั้งหมด หรือไม่ลบเลยสักหมวด
//     (ถ้าส่งหมวดซ้ำ จะลบแค่ครั้งเดียว)
//  3. ถ้าลบอุปกรณ์ ส่งเลขช่องใหม่ (ว่าง) ให้บอร์ดทุกคอกทาง MQTT บอร์ดจะได้หยุดส่งค่าของอุปกรณ์ที่ไม่มีแล้ว
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
