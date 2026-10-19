// =====================================================================================
// ไฟล์ handlers/coop_food_consumption.go - API ประมาณอาหารที่แต่ละคอกกินต่อวัน
// =====================================================================================
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetCoopFoodConsumptionHandler (GET /api/foods/coop-consumption)
// คำนวณว่าแต่ละคอกกินอาหารประเภทไหน วันละกี่ กก. ใช้แสดงในหน้าคลังอาหาร
//
// วิธีคำนวณของแต่ละคอก:
//  1. อายุไก่ (สัปดาห์) = (วันนี้ - วันเกิดไก่) / 7
//  2. หาแถวในตารางให้อาหาร (food_age_rule) ที่ช่วงอายุครอบอายุนี้ -> ได้ประเภทอาหารและกรัม/ตัว/วัน
//  3. กก./วัน = จำนวนไก่ x กรัมต่อตัว / 1000 (ทำใน rules.CoopFeed)
//
// คอกที่ไม่มีวันเกิดไก่ คำนวณอายุไม่ได้ จึงคืนค่า 0
//
// ทำไมใช้สูตรเดียวกับการตัดสต็อกจริง (ComputeDailyFoodConsumption):
// ตัวเลขที่ผู้ใช้เห็นในหน้าจอจะตรงกับที่ระบบตัดจริงทุกเช้า
// (ไม่มีเซนเซอร์วัดอาหารรายคอก ตัวเลขนี้จึงเป็นค่าประมาณจากจำนวนไก่)
func GetCoopFoodConsumptionHandler(w http.ResponseWriter, r *http.Request) {
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

	coops, err := database.GetAllCoops(userID)
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to fetch coops: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to fetch coops", http.StatusInternalServerError)
		return
	}

	rules := loadFoodAgeRules(userID)
	now := time.Now()
	result := make([]models.CoopFoodConsumption, 0, len(coops))
	for _, coop := range coops {
		if coop.Birthday.IsZero() {
			result = append(result, models.CoopFoodConsumption{
				CoopID:            coop.CoopID,
				NameCoop:          coop.NameCoop,
				Amount:            coop.Amount,
				AgeWeeks:          0,
				FoodType:          "",
				EstimatedKgPerDay: 0,
			})
			continue
		}

		ageWeeks := int(now.Sub(coop.Birthday).Hours() / 24 / 7)
		if ageWeeks < 0 {
			ageWeeks = 0
		}
		foodType, kgPerDay, rule := rules.CoopFeed(ageWeeks, coop.Amount)
		var minWeeks, maxWeeks int
		var gramsPerBird float64
		if rule != nil {
			minWeeks, maxWeeks, gramsPerBird = rule.MinWeeks, rule.MaxWeeks, rule.GramsPerBird
		}

		result = append(result, models.CoopFoodConsumption{
			CoopID:            coop.CoopID,
			NameCoop:          coop.NameCoop,
			Amount:            coop.Amount,
			AgeWeeks:          ageWeeks,
			FoodType:          foodType,
			EstimatedKgPerDay: kgPerDay,
			AgeMinWeeks:       minWeeks,
			AgeMaxWeeks:       maxWeeks,
			GramsPerBird:      gramsPerBird,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Estimated food consumption for %d coops", r.Method, r.RequestURI, http.StatusOK, len(result))
	json.NewEncoder(w).Encode(result)
}
