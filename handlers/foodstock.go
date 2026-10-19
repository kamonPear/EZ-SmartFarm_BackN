// =====================================================================================
// ไฟล์ handlers/foodstock.go - API คลังอาหาร และ "ระบบตัดสต็อกอาหารประจำวัน"
// =====================================================================================
//
// หัวใจของไฟล์นี้คือการตัดสต็อกอัตโนมัติ:
//
//	อาหารที่ใช้ต่อวัน = ผลรวมของทุกคอก (จำนวนไก่ x กรัม/ตัว/วัน ตามช่วงอายุ ในตารางให้อาหาร)
//	แยกคิดตามประเภทอาหาร (เม็ดเล็ก / เม็ดใหญ่)
//
// การตัดสต็อกเกิดได้ 4 ทาง (ทุกทางใช้สูตรเดียวกัน ComputeDailyFoodConsumptionByCoop):
//  1. cron 18:00 น. (scheduler/cron.go)          -> RunDailyDeductionForAllUsers
//  2. ตัวปลุกงานจากภายนอก เช่น cron-job.org      -> RunScheduledFoodDeductionHandler
//  3. ตัดตอนผู้ใช้เปิดแอปหลัง 18:00 น. (ถ้าวันนี้ยังไม่ตัด) -> RunCatchUpDeduction
//  4. ผู้ใช้กดปุ่ม "ตัดสต็อก" เอง (ได้ทุกเวลา)      -> ForceDeductStockHandler
//
// กฎเวลา: การตัด "อัตโนมัติ" (ทาง 1-3) ทำได้ตั้งแต่ 18:00 น. ของวันนั้นเป็นต้นไปเท่านั้น และวันละครั้ง
// ก่อน 18:00 น. ไม่มีการตัดอัตโนมัติเลย เช่น เพิ่งกรอกข้อมูลอาหารตอนบ่าย สต็อกจะยังไม่ถูกหักทันที
// ถ้าวันนั้นกดตัดเองไปแล้ว รอบ 18:00 น. จะข้าม (ถือว่าวันนี้ให้อาหารแล้ว ไม่หักซ้ำ)
//
// ทำไมต้องมีหลายทาง: Render แพลนฟรีพักเซิร์ฟเวอร์เมื่อไม่มีคนใช้ ตอน 18:00 น. cron อาจไม่ได้ทำงาน
//
// ทุกครั้งที่ตัดสำเร็จ จะบันทึกยอดรายคอกลง food_distribution ด้วย
// ใช้แสดงสรุปว่าคอกไหนกินไปเท่าไร และใช้เช็กว่า "วันนี้ตัดไปแล้วหรือยัง" (กันตัดซ้ำ)
// GET /api/foods แยกที่ handleFoodsGet ใน routes.go: มี ?id= -> GetFoodstockHandler, ไม่มี -> GetAllFoodstocksHandler
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetFoodstockHandler (GET /api/foods?id=1) ดึงสต็อกแถวเดียว (เฉพาะของผู้ใช้) ไม่เจอ -> 404
func GetFoodstockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing foodstock id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing foodstock id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid foodstock id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid foodstock id", http.StatusBadRequest)
		return
	}

	foodstock, err := database.GetFoodstockByID(id, userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Foodstock not found)", r.Method, r.RequestURI, http.StatusNotFound)
		http.Error(w, "Foodstock not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved foodstock ID: %d", r.Method, r.RequestURI, http.StatusOK, foodstock.FoodID)
	json.NewEncoder(w).Encode(foodstock)
}

// GetAllFoodstocksHandler (GET /api/foods) ดึงสต็อกทุกประเภทของผู้ใช้ (หน้าคลังอาหาร)
// เรียก RunCatchUpDeduction ก่อน (ไม่ใช่ goroutine) เพื่อให้ตัดสต็อกที่ค้างเสร็จก่อน
// ตัวเลขที่ผู้ใช้เห็นจึงเป็นยอดล่าสุดหลังตัดแล้ว
func GetAllFoodstocksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	RunCatchUpDeduction(userID)

	foodstocks, err := database.GetAllFoodstocks(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch foodstocks: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch foodstocks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d foodstocks", r.Method, r.RequestURI, http.StatusOK, len(foodstocks))
	json.NewEncoder(w).Encode(foodstocks)
}

// UpdateFoodstockHandler (PUT /api/foods?id=1) แก้ยอดคงเหลือให้ตรงกับของจริงที่นับได้
// ไม่เจอ/ไม่ใช่ของผู้ใช้ -> 404
func UpdateFoodstockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing foodstock id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing foodstock id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid foodstock id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid foodstock id", http.StatusBadRequest)
		return
	}

	var req models.UpdateFoodstockRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	foodstock, err := database.UpdateFoodstock(id, &req, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Foodstock not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Foodstock not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to update foodstock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update foodstock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Updated foodstock ID: %d", r.Method, r.RequestURI, http.StatusOK, foodstock.FoodID)
	json.NewEncoder(w).Encode(foodstock)
}

// UpdateFoodMinQuantityHandler (PUT /api/foods/min-quantity) ตั้งปริมาณใกล้หมดสำหรับแจ้งเตือน body: {"min_quantity": 50}
// ค่าเดียวใช้ร่วมกันทุกประเภทอาหาร และตั้งได้ครั้งเดียว
//  - ต้องมากกว่า 0 ไม่งั้น 400
//  - เคยตั้งแล้ว -> 409 (ล็อกแล้ว แก้ไขไม่ได้)
//  - ยังไม่เคยนำเข้าอาหาร (ไม่มีแถวสต็อก) -> 404 ให้นำเข้าก่อน
func UpdateFoodMinQuantityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.UpdateFoodMinQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.MinQuantity <= 0 {
		http.Error(w, "min_quantity ต้องมากกว่า 0", http.StatusBadRequest)
		return
	}

	// ค่าเดียวใช้ร่วมกันทุกประเภทอาหาร (food_type ในคำขอไม่จำเป็น ไม่ได้ใช้แล้ว)
	// ตั้งได้ครั้งเดียว หลังจากนั้นล็อก แก้ไขไม่ได้
	if err := database.SetFoodMinQuantity(userID, req.MinQuantity); err != nil {
		if errors.Is(err, database.ErrMinQuantityLocked) {
			http.Error(w, "ตั้งปริมาณอาหารใกล้หมดไว้แล้ว แก้ไขไม่ได้", http.StatusConflict)
			return
		}
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "ยังไม่เคยนำเข้าอาหารประเภทนี้ กรุณานำเข้าสต็อกก่อน", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to set min quantity: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to update min quantity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Set min quantity (all food types) to %.2f", r.Method, r.RequestURI, http.StatusOK, req.MinQuantity)
	json.NewEncoder(w).Encode(map[string]float64{"min_quantity": req.MinQuantity})
}

// DeleteFoodstockHandler (DELETE /api/foods?id=1) ลบแถวสต็อก ไม่เจอ -> 404
func DeleteFoodstockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[%s] %s - %d (Missing foodstock id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Missing foodstock id parameter", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[%s] %s - %d (Invalid foodstock id)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid foodstock id", http.StatusBadRequest)
		return
	}

	if err := database.DeleteFoodstock(id, userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			log.Printf("[%s] %s - %d (Foodstock not found)", r.Method, r.RequestURI, http.StatusNotFound)
			http.Error(w, "Foodstock not found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] %s - %d (Failed to delete foodstock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to delete foodstock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Deleted foodstock ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
	json.NewEncoder(w).Encode(map[string]string{"message": "Foodstock deleted successfully"})
}

// GetFoodHistoryHandler (GET /api/food_history) ดึงประวัติการนำเข้าอาหาร (ตาราง importfood)
// ผลเหมือน GET /api/importfoods เก็บไว้ให้แอปรุ่นเดิมที่เรียก path นี้
func GetFoodHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	history, err := database.GetAllImportFood(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch food history: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch food history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Retrieved %d food history records", r.Method, r.RequestURI, http.StatusOK, len(history))
	json.NewEncoder(w).Encode(history)
}

// loadFoodAgeRules ดึงตารางให้อาหารของผู้ใช้จาก DB ถ้าอ่านไม่ได้คืนตารางว่าง
// ตารางว่าง = ไม่มีคอกไหนถูกคำนวณ/ตัดสต็อก (ไม่มีค่าตายตัวในโค้ด ผู้ใช้ต้องตั้งเอง)
// ทำไมคืนตารางว่างแทน error: งานตัดสต็อกของผู้ใช้คนอื่นจะได้ทำต่อได้ ไม่พังทั้งรอบ
// ถูกเรียกจาก: ฟังก์ชันตัดสต็อกทุกตัวในไฟล์นี้ และ coop_food_consumption.go
func loadFoodAgeRules(userID int) models.FoodRules {
	rules, err := database.GetFoodAgeRules(userID)
	if err != nil {
		return models.FoodRules{}
	}
	return rules
}

// ComputeDailyFoodConsumptionByCoop คำนวณ กก./วัน ของอาหารประเภท foodType แยกรายคอก
// วนทุกคอก:
//  1. ไม่มีวันเกิดไก่ -> ข้าม (ไม่รู้อายุ)
//  2. อายุ (สัปดาห์) = (ตอนนี้ - วันเกิด) / 7
//  3. rules.CoopFeed หาช่วงอายุที่ตรง -> ได้ประเภทอาหาร และ กก. = จำนวนไก่ x กรัม / 1000
//  4. เก็บเฉพาะคอกที่กินอาหารประเภทนี้อยู่ และ กก. > 0
//
// คืนรายการ {coop_id, kg_given} ใช้ได้ 2 อย่าง:
//   - รวมกันเป็นยอดที่ต้องตัดสต็อก
//   - บันทึกลง food_distribution ให้รู้ว่าคอกไหนกินไปเท่าไร
func ComputeDailyFoodConsumptionByCoop(coops []models.Coop, foodType string, rules models.FoodRules) []models.FoodDistributionItem {
	now := time.Now()
	items := make([]models.FoodDistributionItem, 0, len(coops))
	for _, coop := range coops {
		if coop.Birthday.IsZero() {
			continue // ยังไม่ทราบอายุ ไม่รวมในการคำนวณ
		}
		ageWeeks := int(now.Sub(coop.Birthday).Hours() / 24 / 7)
		if ageWeeks < 0 {
			ageWeeks = 0
		}
		coopFoodType, kg, _ := rules.CoopFeed(ageWeeks, coop.Amount)
		if coopFoodType != foodType {
			continue
		}
		if kg <= 0 {
			continue
		}
		items = append(items, models.FoodDistributionItem{CoopID: coop.CoopID, KgGiven: kg})
	}
	return items
}

// ComputeDailyFoodConsumption รวมยอด กก./วัน ของทุกคอก (ผลรวมของ ComputeDailyFoodConsumptionByCoop)
// สำหรับที่ต้องการแค่ยอดรวม ไม่ต้องการรายคอก
func ComputeDailyFoodConsumption(coops []models.Coop, foodType string, rules models.FoodRules) float64 {
	var total float64
	for _, item := range ComputeDailyFoodConsumptionByCoop(coops, foodType, rules) {
		total += item.KgGiven
	}
	return total
}

// DailyDeductionHour เวลาตัดสต็อกอาหารอัตโนมัติประจำวัน (ชั่วโมง เวลาไทย) = 18:00 น.
// ใช้ทั้งตั้ง cron (scheduler/cron.go) และเช็กว่า "ถึงเวลาตัดของวันนี้หรือยัง"
const DailyDeductionHour = 18

// IsDailyDeductionTime ถึงเวลาตัดสต็อกอัตโนมัติของวันนี้แล้วหรือยัง (18:00 น. เวลาไทยเป็นต้นไป)
// ก่อนเวลานี้ทุกทางอัตโนมัติจะไม่ตัด กันสต็อกถูกหักทันทีตอนเพิ่งกรอกข้อมูลอาหาร/เปิดแอปตอนกลางวัน
func IsDailyDeductionTime(now time.Time) bool {
	return now.In(models.BangkokLocation()).Hour() >= DailyDeductionHour
}

// deductTodayForUser ตัดสต็อกอาหารประจำวัน "ของวันนี้" ให้ผู้ใช้คนเดียว (ใช้ร่วมกันทุกทางอัตโนมัติ)
//
// ทำแยกแต่ละประเภทอาหาร:
//  1. วันนี้มีบันทึกการตัด (food_distribution) ของประเภทนี้แล้ว -> ข้าม กันตัดซ้ำ
//     (ตัดไปแล้วจาก cron / ตัวปลุกจากภายนอก / เปิดแอปรอบก่อน / ผู้ใช้กดตัดเอง)
//  2. คำนวณยอดรายคอก รวมเป็น 0 (ไม่มีคอกกินประเภทนี้ หรือยังไม่ได้ตั้งตารางให้อาหาร) -> ข้าม
//  3. ตัดสต็อกแบบทั้งหมดหรือไม่ตัดเลย ไม่พอ/ยังไม่มีสต็อก -> ไม่ตัด (ลองใหม่ครั้งถัดไปของวันนี้)
//  4. ตัดสำเร็จ -> บันทึกยอดรายคอก (ทำให้รอบถัดไปของวันนี้รู้ว่าตัดแล้ว)
//
// คืน true ถ้าตัดสำเร็จอย่างน้อย 1 ประเภท  tag ใช้นำหน้า log ให้รู้ว่ามาจากทางไหน
func deductTodayForUser(userID int, coops []models.Coop, tag string) bool {
	rules := loadFoodAgeRules(userID)
	deductedAny := false
	for _, foodType := range []string{models.FoodTypeSmallPellet, models.FoodTypeLargePellet} {
		lastDate, found, err := database.GetLatestDistributionDate(userID, foodType)
		if err != nil {
			log.Printf("❌ [%s] ตรวจสอบวันตัดสต็อกล่าสุดของ user %d ล้มเหลว: %v\n", tag, userID, err)
			continue
		}
		if found && models.DateKey(lastDate) == models.DateKey(time.Now()) {
			continue // วันนี้ตัดไปแล้ว
		}

		items := ComputeDailyFoodConsumptionByCoop(coops, foodType, rules)
		var amount float64
		for _, item := range items {
			amount += item.KgGiven
		}
		if amount <= 0 {
			continue // ไม่มีคอกไหนกำลังกินอาหารประเภทนี้อยู่
		}

		deducted, shortfall, err := database.DeductFoodstockByType(foodType, userID, amount)
		if err != nil {
			log.Printf("❌ [%s] ตัดสต็อก %s ของ user %d ล้มเหลว: %v\n", tag, foodType, userID, err)
			continue
		}
		if shortfall > 0 {
			// สต็อกไม่พอ (หรือยังไม่มีข้อมูลอาหารเลย) - ไม่ตัดอะไร ไม่มีบันทึกของวันนี้ จึงจะลองใหม่
			// ในการเรียกถัดไปของวันนี้ (เช่นหลังเติมสต็อก)
			log.Printf("⚠️ [%s] สต็อก %s ของ user %d ไม่พอ ต้องการ %.2f kg (ขาด %.2f kg) ไม่ได้ตัดสต็อก\n", tag, foodType, userID, amount, shortfall)
			continue
		}
		if _, err := database.CreateFoodDistributionBatch(foodType, items, userID); err != nil {
			log.Printf("❌ [%s] บันทึก breakdown ของ user %d ล้มเหลว (ตัดสต็อกไปแล้ว): %v\n", tag, userID, err)
		}
		log.Printf("✅ [%s] ตัดสต็อก %s %.2f กก. ให้ user %d\n", tag, foodType, deducted, userID)
		deductedAny = true
	}
	return deductedAny
}

// RunDailyDeductionForAllUsers ตัดสต็อกอาหารประจำวันให้ "ทุกผู้ใช้ในระบบ"
// คืนค่า: จำนวนผู้ใช้ที่ถูกตัดสต็อกสำเร็จ
//
// ขั้นตอน:
//  1. ยังไม่ถึง 18:00 น. -> ไม่ทำอะไร (เช่น ตัวปลุกจากภายนอกตั้งเวลาผิด) คืน 0
//  2. ดึงคอกทั้งหมดจัดกลุ่มตามผู้ใช้ (GetAllCoopsGroupedByUser) query ครั้งเดียว
//  3. แต่ละผู้ใช้เรียก deductTodayForUser (ข้ามประเภทที่วันนี้ตัดไปแล้ว จึงเรียกซ้ำในวันเดียวได้ไม่หักซ้ำ)
//     error ของผู้ใช้คนหนึ่งแค่ log แล้วไปต่อ ผู้ใช้คนอื่นยังได้ตัด
//
// ทำไมแยกเป็นฟังก์ชันกลาง: cron และตัวปลุกงานจากภายนอกใช้ร่วมกัน แก้ที่เดียวพอ
// ถูกเรียกจาก: scheduler/cron.go และ RunScheduledFoodDeductionHandler
func RunDailyDeductionForAllUsers() (deductedUsers int, err error) {
	if !IsDailyDeductionTime(time.Now()) {
		log.Printf("⏳ [Food] ยังไม่ถึงเวลาตัดสต็อกอัตโนมัติ (%02d:00 น.) ข้ามรอบนี้\n", DailyDeductionHour)
		return 0, nil
	}

	coopsByUser, err := database.GetAllCoopsGroupedByUser()
	if err != nil {
		return 0, err
	}

	for userID, coops := range coopsByUser {
		if deductTodayForUser(userID, coops, "Food") {
			deductedUsers++
		}
	}

	return deductedUsers, nil
}

// RunScheduledFoodDeductionHandler (POST /api/system/food-deduction/run?key=xxx)
// ให้บริการตั้งเวลาภายนอก (เช่น cron-job.org) ยิง URL นี้ทุกเช้า เพื่อตัดสต็อกให้ทุกผู้ใช้
// ทำไมต้องมี: Render แพลนฟรีพักเซิร์ฟเวอร์ ถ้าทั้งวันไม่มีคนเปิดแอป cron ในตัวและ RunCatchUpDeduction
// จะไม่ได้ทำงานเลย request จากภายนอกนี้เองที่ปลุกเซิร์ฟเวอร์ขึ้นมา
// ป้องกันด้วย SCHEDULED_TASK_KEY (auth.RequireScheduledTaskKey) แทน JWT เพราะผู้เรียกไม่ใช่ผู้ใช้
// ต้องตั้งให้เรียกหลัง 18:00 น. (เรียกก่อนเวลาจะไม่ตัดอะไร) เรียกซ้ำในวันเดียวได้ ไม่หักซ้ำ
// ตอบกลับจำนวนผู้ใช้ที่ถูกตัด
func RunScheduledFoodDeductionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deductedUsers, err := RunDailyDeductionForAllUsers()
	if err != nil {
		log.Printf("[%s] %s - %d (Scheduled deduction failed: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to run scheduled deduction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ ตัดสต็อกอาหารประจำวัน (external trigger) เสร็จสิ้น (%d ผู้ใช้)", r.Method, r.RequestURI, http.StatusOK, deductedUsers)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "ตัดสต็อกอาหารประจำวันสำเร็จ",
		"deducted_users": deductedUsers,
	})
}

// RunCatchUpDeduction ตัดสต็อกของวันนี้ให้ผู้ใช้คนเดียว ตอนผู้ใช้เปิดแอป ถ้าเลย 18:00 น. แล้วแต่ยังไม่ได้ตัด
//
// ทำไมต้องมี: cron 18:00 น. ทำงานได้เฉพาะตอนเซิร์ฟเวอร์ตื่นอยู่ Render แพลนฟรีพักเซิร์ฟเวอร์
// หลังไม่มีคนใช้ ~15 นาที ถ้าตอน 18:00 น. ไม่มีใครใช้ cron จะพลาดโดยไม่มี error ให้เห็น
// จึงใช้จังหวะที่ผู้ใช้เปิดแอปหลัง 18:00 น. เป็นจุดสำรอง
//
// ก่อน 18:00 น. ไม่ตัดเลย - เช่นคอกมีไก่แต่เพิ่งกรอกข้อมูลอาหารตอนกลางวัน สต็อกจะไม่ถูกหักทันที
// ขั้นตอนตัดจริงอยู่ใน deductTodayForUser (ข้ามถ้าวันนี้ตัดไปแล้ว)
// ถูกเรียกจาก: GetAllFoodstocksHandler (หน้าคลังอาหาร) และ GetAllCoopsHandler ใน coop.go (หน้าหลัก)
func RunCatchUpDeduction(userID int) {
	if !IsDailyDeductionTime(time.Now()) {
		return
	}
	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("❌ [CatchUp] ดึงข้อมูลคอกของ user %d ล้มเหลว: %v\n", userID, err)
		return
	}
	deductTodayForUser(userID, coops, "CatchUp")
}

// ForceDeductStockHandler (POST /api/foodstocks/force-deduct) ผู้ใช้กดตัดสต็อกเอง
// body: {"food_type": "เม็ดเล็ก"} ต้องระบุประเภทเสมอ
// ยอดที่ตัดคำนวณจากจำนวนไก่จริงด้วยสูตรเดียวกับการตัดอัตโนมัติ (ไม่ใช่ค่าตายตัว)
// ผลลัพธ์ที่เป็นไปได้ (ตอบ 200 ทุกกรณี พร้อมข้อความให้แสดง):
//   - ไม่มีคอกกินประเภทนี้ -> "ไม่มีอะไรให้ตัด" deducted = 0
//   - สต็อกไม่พอ -> ไม่ตัด บอกว่ามีเท่าไร ต้องการเท่าไร ขาดเท่าไร
//   - สำเร็จ -> บันทึกยอดรายคอก แล้วบอกว่าตัดไปกี่ กก.
//     (ถ้าบันทึกยอดรายคอกล้ม แค่ log เพราะสต็อกถูกตัดไปแล้ว ไม่ควรตอบว่าล้มทั้งคำขอ)
func ForceDeductStockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.DeductFoodstockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[%s] %s - %d (Invalid request body)", r.Method, r.RequestURI, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !models.IsValidFoodType(req.FoodType) {
		log.Printf("[%s] %s - %d (Invalid food_type: %q)", r.Method, r.RequestURI, http.StatusBadRequest, req.FoodType)
		http.Error(w, "ต้องระบุ food_type เป็น 'เม็ดเล็ก' หรือ 'เม็ดใหญ่'", http.StatusBadRequest)
		return
	}

	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch coops", http.StatusInternalServerError)
		return
	}

	items := ComputeDailyFoodConsumptionByCoop(coops, req.FoodType, loadFoodAgeRules(userID))
	var amount float64
	for _, item := range items {
		amount += item.KgGiven
	}
	if amount <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ ไม่มีคอกที่กินอาหาร %s อยู่ ไม่มีอะไรให้ตัด", r.Method, r.RequestURI, http.StatusOK, req.FoodType)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("ไม่มีคอกที่กำลังกินอาหาร%sอยู่ ไม่มีอะไรให้ตัดสต็อก", req.FoodType),
			"food_type": req.FoodType,
			"deducted":  0,
		})
		return
	}

	deducted, shortfall, err := database.DeductFoodstockByType(req.FoodType, userID, amount)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to deduct stock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to deduct stock", http.StatusInternalServerError)
		return
	}

	// สต็อกไม่พอสำหรับยอดที่ต้องการเต็มจำนวน (ตัดแบบทั้งหมดหรือไม่เลย) - ไม่มีอะไร
	// ถูกตัดออกจากสต็อกจริง เลยไม่มีอะไรให้บันทึก breakdown ด้วย
	if shortfall > 0 {
		message := fmt.Sprintf(
			"สต็อกอาหาร%sเหลือไม่พอ มีอยู่ %.2f กก. แต่ต้องการ %.2f กก. (ขาดอีก %.2f กก.) ยังไม่ได้ตัดสต็อก กรุณาเติมสต็อกก่อน",
			req.FoodType, amount-shortfall, amount, shortfall,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ %s", r.Method, r.RequestURI, http.StatusOK, message)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   message,
			"food_type": req.FoodType,
			"deducted":  0,
			"shortfall": shortfall,
		})
		return
	}

	// บันทึก breakdown รายคอกไว้ด้วย (เหมือนตอนกรอกเองสมัยก่อน) ให้หน้า "สรุปผล
	// อาหารแต่ละประเภท" ยังเห็นว่าคอกไหนกินไปเท่าไรได้ แม้ตัดสต็อกแบบไม่กรอกเอง -
	// สต็อกถูกตัดสำเร็จไปแล้วข้างบน พลาดตรงนี้แค่ log ไว้ ไม่ทำให้ทั้งคำขอ fail
	if _, err := database.CreateFoodDistributionBatch(req.FoodType, items, userID); err != nil {
		log.Printf("[%s] %s - (บันทึก breakdown รายคอกไม่สำเร็จ แต่ตัดสต็อกไปแล้ว: %v)", r.Method, r.RequestURI, err)
	}

	message := fmt.Sprintf("หักสต็อกอาหาร%s %.2f กิโลกรัม สำเร็จ", req.FoodType, deducted)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ %s", r.Method, r.RequestURI, http.StatusOK, message)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   message,
		"food_type": req.FoodType,
		"deducted":  deducted,
		"shortfall": shortfall,
	})
}
