package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetVaccineHandler retrieves vaccine records
// GET /api/vaccines
// Query parameters:
// - id: Get vaccine by vaccine ID
// - coop_id: Get vaccines by coop ID
// - (no params): Get all vaccines
func GetVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	vaccineID := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	// Get vaccine by ID
	if vaccineID != "" {
		id, err := strconv.Atoi(vaccineID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid vaccine ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid vaccine ID format", http.StatusBadRequest)
			return
		}

		vaccine, err := database.GetVaccineByID(id)
		if err != nil {
			log.Printf("[%s] %s - %d (Vaccine not found: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
			http.Error(w, "Vaccine not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Retrieved vaccine ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
		json.NewEncoder(w).Encode(vaccine)
		return
	}

	// Get vaccines by coop ID
	if coopID != "" {
		id, err := strconv.Atoi(coopID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid coop ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		vaccines, err := database.GetVaccinesByCoopID(id)
		if err != nil {
			log.Printf("[%s] %s - %d (Failed to retrieve vaccines: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
			http.Error(w, "Failed to retrieve vaccines", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Retrieved %d vaccines for coop ID: %d", r.Method, r.RequestURI, http.StatusOK, len(vaccines), id)
		json.NewEncoder(w).Encode(vaccines)
		return
	}

	// Get all vaccines
	vaccines, err := database.GetAllVaccines()
	if err != nil {
		log.Printf("[%s] %s - %d (Failed to retrieve vaccines: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
		http.Error(w, "Failed to retrieve vaccines", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved %d vaccines", r.Method, r.RequestURI, http.StatusOK, len(vaccines))
	json.NewEncoder(w).Encode(vaccines)
}

// DeleteVaccineHandler deletes a vaccine record
// DELETE /api/vaccines
// Query parameters:
// - id: Delete vaccine by vaccine ID
// - coop_id: Delete all vaccines by coop ID
func DeleteVaccineHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	vaccineID := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	// Delete vaccine by ID
	if vaccineID != "" {
		id, err := strconv.Atoi(vaccineID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid vaccine ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid vaccine ID format", http.StatusBadRequest)
			return
		}

		err = database.DeleteVaccine(id)
		if err != nil {
			log.Printf("[%s] %s - %d (Failed to delete vaccine: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
			http.Error(w, "Failed to delete vaccine", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Deleted vaccine ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
		json.NewEncoder(w).Encode(map[string]string{"message": "Vaccine deleted successfully"})
		return
	}

	// Delete vaccines by coop ID
	if coopID != "" {
		id, err := strconv.Atoi(coopID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid coop ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		err = database.DeleteVaccinesByCoopID(id)
		if err != nil {
			log.Printf("[%s] %s - %d (Failed to delete vaccines: %v)", r.Method, r.RequestURI, http.StatusInternalServerError, err)
			http.Error(w, "Failed to delete vaccines", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Deleted all vaccines for coop ID: %d", r.Method, r.RequestURI, http.StatusOK, id)
		json.NewEncoder(w).Encode(map[string]string{"message": "All vaccines for coop deleted successfully"})
		return
	}

	log.Printf("[%s] %s - %d (Missing vaccine_id or coop_id parameter)", r.Method, r.RequestURI, http.StatusBadRequest)
	http.Error(w, "Missing vaccine_id or coop_id parameter", http.StatusBadRequest)
}

// GetRecommendedVaccinesHandler retrieves recommended vaccines based on chicken age
// GET /api/vaccines/recommended
// Query parameters:
// - coop_id: Get recommended vaccines for a specific coop
func GetRecommendedVaccinesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("[%s] %s - %d (Method not allowed)", r.Method, r.RequestURI, http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	coopID := r.URL.Query().Get("coop_id")

	// Get recommended vaccines for specific coop
	if coopID != "" {
		id, err := strconv.Atoi(coopID)
		if err != nil {
			log.Printf("[%s] %s - %d (Invalid coop ID format)", r.Method, r.RequestURI, http.StatusBadRequest)
			http.Error(w, "Invalid coop ID format", http.StatusBadRequest)
			return
		}

		// Get the coop to retrieve birthday and calculate age
		coop, err := database.GetCoopByID(id)
		if err != nil {
			log.Printf("[%s] %s - %d (Coop not found: %v)", r.Method, r.RequestURI, http.StatusNotFound, err)
			http.Error(w, "Coop not found", http.StatusNotFound)
			return
		}

		// Calculate chicken age in days
		ageInDays := models.CalculateChickenAge(coop.Birthday)
		log.Printf("DEBUG: Coop ID %d - Birthday: %v, Current Age: %d days", id, coop.Birthday, ageInDays)

		// Get recommended vaccines based on age
		recommendedVaccines := models.GetVaccineScheduleByAge(ageInDays)
		log.Printf("DEBUG: Found %d recommended vaccines for coop ID %d (age %d days)", len(recommendedVaccines), id, ageInDays)

		w.WriteHeader(http.StatusOK)
		log.Printf("[%s] %s - %d ✓ Retrieved recommended vaccines for coop ID: %d (age: %d days)", r.Method, r.RequestURI, http.StatusOK, id, ageInDays)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"coop_id":              id,
			"birthday":             coop.Birthday,
			"current_age_days":     ageInDays,
			"recommended_vaccines": recommendedVaccines,
		})
		return
	}

	// Get all vaccine schedules (fixed data)
	allSchedules := models.GetAllVaccineSchedules()
	w.WriteHeader(http.StatusOK)
	log.Printf("[%s] %s - %d ✓ Retrieved all vaccine schedules", r.Method, r.RequestURI, http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schedules": allSchedules,
	})
}
