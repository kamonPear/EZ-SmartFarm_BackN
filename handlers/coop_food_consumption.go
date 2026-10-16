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
// Each coop's food type is decided by the age of its chickens (the owner's FoodAgeConfig),
// and the estimate is a direct headcount calculation (coop.Amount x
// configured grams per bird) - the same formula ComputeDailyFoodConsumption
// uses for the real automatic deduction, so what's shown here always matches what actually
// gets deducted (no per-coop feed sensor exists, so this is still an estimate, just no
// longer a proportional split of an artificial fixed pool).
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

	cfg := loadFoodAgeConfig(userID)
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
		foodType, kgPerDay := cfg.CoopFeed(ageWeeks, coop.Amount)
		minWeeks, maxWeeks := cfg.AgeRangeFor(foodType)

		result = append(result, models.CoopFoodConsumption{
			CoopID:            coop.CoopID,
			NameCoop:          coop.NameCoop,
			Amount:            coop.Amount,
			AgeWeeks:          ageWeeks,
			FoodType:          foodType,
			EstimatedKgPerDay: kgPerDay,
			AgeMinWeeks:       minWeeks,
			AgeMaxWeeks:       maxWeeks,
			GramsPerBird:      cfg.GramsPerBirdFor(foodType),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s - %d ✓ Estimated food consumption for %d coops", r.Method, r.RequestURI, http.StatusOK, len(result))
	json.NewEncoder(w).Encode(result)
}
