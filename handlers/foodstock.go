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

// GetFoodstockHandler retrieves a single foodstock by ID
// GET /api/foodstocks?id=1
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

// GetAllFoodstocksHandler retrieves all foodstocks owned by the caller
// GET /api/foodstocks
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

// UpdateFoodstockHandler manually corrects the current stock total
// PUT /api/foodstocks?id=1
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

// DeleteFoodstockHandler deletes a foodstock
// DELETE /api/foodstocks?id=1
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

// GetFoodHistoryHandler retrieves the caller's history of food imports (importfood table)
// GET /api/food_history
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

// FoodConsumptionKgPerBirdPerDay คือกก./ตัว/วัน ที่ไก่ 1 ตัวกินอาหารแต่ละประเภท
// (ประมาณการตามช่วงอายุ - ไก่เล็กกินน้อยกว่าไก่โต) ใช้คูณกับจำนวนไก่จริงของแต่ละ
// คอก (coop.Amount) เพื่อคำนวณยอดตัดสต็อกจริงของผู้ใช้แต่ละคน แทนที่ค่าคงที่ตายตัว
// แบบเดิม (DailyDeductAmounts) ที่ตัดเท่ากันทุกบัญชีไม่ว่าจะเลี้ยงไก่กี่ตัวก็ตาม
var FoodConsumptionKgPerBirdPerDay = map[string]float64{
	models.FoodTypeSmallPellet: 0.06, // ไก่อายุ 0-14 สัปดาห์ ~60 กรัม/ตัว/วัน
	models.FoodTypeLargePellet: 0.13, // ไก่อายุ 14 สัปดาห์ขึ้นไป ~130 กรัม/ตัว/วัน
}

// ComputeDailyFoodConsumptionByCoop คืนยอดกก./วันของอาหารประเภท foodType แยกเป็น
// รายคอก (เฉพาะคอกที่กำลังกินอาหารประเภทนี้อยู่จริง) - อิงจากจำนวนไก่จริงต่อคอก
// (coop.Amount) คูณอัตรากก./ตัว/วันของช่วงอายุคอกนั้น (ตัดสินจาก Birthday ว่าอยู่
// ช่วงเม็ดเล็ก/เม็ดใหญ่) ใช้ทั้งเป็นยอดตัดสต็อกรวม (sum ของผลลัพธ์นี้) และเป็น
// breakdown ที่บันทึกลง food_distribution ทุกครั้งที่ตัดสต็อกอัตโนมัติ เพื่อให้หน้า
// "สรุปผลอาหารแต่ละประเภท" ยังเห็นว่าคอกไหนกินไปเท่าไรได้ แม้ไม่มีใครกรอกเอง
func ComputeDailyFoodConsumptionByCoop(coops []models.Coop, foodType string) []models.FoodDistributionItem {
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
		if models.FoodTypeForAgeWeeks(ageWeeks) != foodType {
			continue
		}
		kg := float64(coop.Amount) * FoodConsumptionKgPerBirdPerDay[foodType]
		if kg <= 0 {
			continue
		}
		items = append(items, models.FoodDistributionItem{CoopID: coop.CoopID, KgGiven: kg})
	}
	return items
}

// ComputeDailyFoodConsumption รวมยอดกก./วัน ของอาหารประเภท foodType ที่คอกทั้งหมด
// ที่ส่งมา (ของผู้ใช้คนเดียว) กินจริง - ผลรวมของ ComputeDailyFoodConsumptionByCoop
// ใช้ร่วมกันทั้งตัวตัดสต็อกแมนนวล (ForceDeductStockHandler), Cron ตัดสต็อกประจำวัน
// (scheduler.SetupJobs) และตัวประมาณการต่อคอกที่โชว์ในหน้าคลังอาหาร (GetCoopFoodConsumptionHandler)
// เพื่อให้ตัวเลขตรงกันทุกจุด
func ComputeDailyFoodConsumption(coops []models.Coop, foodType string) float64 {
	var total float64
	for _, item := range ComputeDailyFoodConsumptionByCoop(coops, foodType) {
		total += item.KgGiven
	}
	return total
}

// RunCatchUpDeduction ตรวจว่า userID ถูกตัดสต็อกอาหารประจำวันของวันนี้ (ตามปฏิทินไทย)
// ไปแล้วหรือยัง ถ้ายัง ตัดให้ทันที - มีฟังก์ชันนี้เพราะ cron ตอน 6 โมงเช้า
// (scheduler.SetupJobs) ทำงานได้ก็ต่อเมื่อตัวโปรเซสเซิร์ฟเวอร์ "มีชีวิตอยู่" ณ
// วินาทีนั้นจริงๆ แต่ Render แพลนฟรีจะพักโปรเซสทิ้งหลังไม่มีคนใช้ ~15 นาที แล้วจะตื่น
// ขึ้นมาใหม่ก็ต่อเมื่อมี request เข้ามาเท่านั้น - ถ้าทั้งคืนไม่มีใครเปิดแอปเลย ตัวโปรเซส
// อาจไม่ได้ทำงานอยู่ตอน 06:00 น. พอดี ทำให้ cron ที่ตั้งไว้ไม่มีโอกาสได้รันเลยสักครั้ง
// โดยไม่มี error ให้เห็นที่ไหนเลย เรียกฟังก์ชันนี้ตอนเปิดหน้าคลังอาหาร (ซึ่งเป็น
// request ที่ปลุกเซิร์ฟเวอร์ขึ้นมาแน่ๆ) เพื่อให้มีจุดสำรองที่ตัดสต็อกให้ได้ทุกวัน
// ตราบใดที่ผู้ใช้เปิดแอปอย่างน้อยวันละครั้ง
func RunCatchUpDeduction(userID int) {
	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("❌ [CatchUp] ดึงข้อมูลคอกของ user %d ล้มเหลว: %v\n", userID, err)
		return
	}

	for _, foodType := range []string{models.FoodTypeSmallPellet, models.FoodTypeLargePellet} {
		lastDate, found, err := database.GetLatestDistributionDate(userID, foodType)
		if err != nil {
			log.Printf("❌ [CatchUp] ตรวจสอบวันตัดสต็อกล่าสุดของ user %d ล้มเหลว: %v\n", userID, err)
			continue
		}
		if found && models.DateKey(lastDate) == models.DateKey(time.Now()) {
			continue // ตัดไปแล้ววันนี้ (จาก cron หรือเปิดหน้านี้รอบก่อนของวันเดียวกัน)
		}

		items := ComputeDailyFoodConsumptionByCoop(coops, foodType)
		var amount float64
		for _, item := range items {
			amount += item.KgGiven
		}
		if amount <= 0 {
			continue // ไม่มีคอกไหนกำลังกินอาหารประเภทนี้อยู่
		}

		if err := database.DeductFoodstockByType(foodType, userID, amount); err != nil {
			log.Printf("❌ [CatchUp] ตัดสต็อก %s ของ user %d ล้มเหลว: %v\n", foodType, userID, err)
			continue
		}
		if _, err := database.CreateFoodDistributionBatch(foodType, items, userID); err != nil {
			log.Printf("❌ [CatchUp] บันทึก breakdown ของ user %d ล้มเหลว (ตัดสต็อกไปแล้ว): %v\n", userID, err)
		}
		log.Printf("✅ [CatchUp] ตัดสต็อก %s %.2f กก. ให้ user %d แทน Cron ที่พลาดไป\n", foodType, amount, userID)
	}
}

// ForceDeductStockHandler อนุญาตให้เรียก API เพื่อตัดสต็อกแบบแมนนวล ต้องระบุประเภทอาหารก่อนตัดเสมอ
// ยอดที่ตัดคำนวณจากจำนวนไก่จริงในคอกของผู้เรียก (userID) ที่กำลังกินอาหารประเภทนี้อยู่
// (เหมือนกับยอดที่ Cron ตัดให้อัตโนมัติทุกเที่ยงคืน) ไม่ใช่ค่าคงที่ตายตัวอีกต่อไป
// POST /api/foodstocks/force-deduct  body: {"food_type": "เม็ดเล็ก"}
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

	items := ComputeDailyFoodConsumptionByCoop(coops, req.FoodType)
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

	if err := database.DeductFoodstockByType(req.FoodType, userID, amount); err != nil {
		log.Printf("[%s] %s - %d (Failed to deduct stock: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to deduct stock", http.StatusInternalServerError)
		return
	}

	// บันทึก breakdown รายคอกไว้ด้วย (เหมือนตอนกรอกเองสมัยก่อน) ให้หน้า "สรุปผล
	// อาหารแต่ละประเภท" ยังเห็นว่าคอกไหนกินไปเท่าไรได้ แม้ตัดสต็อกแบบไม่กรอกเอง -
	// สต็อกถูกตัดสำเร็จไปแล้วข้างบน พลาดตรงนี้แค่ log ไว้ ไม่ทำให้ทั้งคำขอ fail
	if _, err := database.CreateFoodDistributionBatch(req.FoodType, items, userID); err != nil {
		log.Printf("[%s] %s - (บันทึก breakdown รายคอกไม่สำเร็จ แต่ตัดสต็อกไปแล้ว: %v)", r.Method, r.RequestURI, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ ตัดสต็อก %s %.2f กก. เรียบร้อยแล้ว", r.Method, r.RequestURI, http.StatusOK, req.FoodType, amount)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   fmt.Sprintf("หักสต็อกอาหาร%s %.2f กิโลกรัม สำเร็จ", req.FoodType, amount),
		"food_type": req.FoodType,
		"deducted":  amount,
	})
}
