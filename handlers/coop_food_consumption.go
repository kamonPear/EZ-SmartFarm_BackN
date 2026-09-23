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

// GetCoopFoodConsumptionHandler estimates how many kg of food each coop eats per day.
// Each coop's food type is decided by the age of its chickens (models.FoodTypeForAgeWeeks).
// The farm only tracks one daily deduction total per food type (see DailyDeductAmounts),
// so a coop's estimated share is that total split proportionally by chicken headcount
// (Amount) among every other coop currently eating the same food type.
// GET /api/foods/coop-consumption
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

	now := time.Now()
	ageWeeksOf := make(map[int]int, len(coops))
	foodTypeOf := make(map[int]string, len(coops))
	amountByType := make(map[string]int)

	for _, coop := range coops {
		if coop.Birthday.IsZero() {
			continue // ยังไม่ทราบอายุ ไม่รวมในการคำนวณสัดส่วน
		}

		ageWeeks := int(now.Sub(coop.Birthday).Hours() / 24 / 7)
		if ageWeeks < 0 {
			ageWeeks = 0
		}
		foodType := models.FoodTypeForAgeWeeks(ageWeeks)

		ageWeeksOf[coop.CoopID] = ageWeeks
		foodTypeOf[coop.CoopID] = foodType
		amountByType[foodType] += coop.Amount
	}

	result := make([]models.CoopFoodConsumption, 0, len(coops))
	for _, coop := range coops {
		foodType, known := foodTypeOf[coop.CoopID]
		if !known {
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

		dailyTotal := DailyDeductAmounts[foodType]
		totalAmount := amountByType[foodType]

		var estimated float64
		if totalAmount > 0 {
			estimated = dailyTotal * float64(coop.Amount) / float64(totalAmount)
		}

		result = append(result, models.CoopFoodConsumption{
			CoopID:            coop.CoopID,
			NameCoop:          coop.NameCoop,
			Amount:            coop.Amount,
			AgeWeeks:          ageWeeksOf[coop.CoopID],
			FoodType:          foodType,
			EstimatedKgPerDay: estimated,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Estimated food consumption for %d coops", r.Method, r.RequestURI, http.StatusOK, len(result))
	json.NewEncoder(w).Encode(result)
}
