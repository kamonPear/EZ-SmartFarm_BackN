package routes

import (
	"net/http"

	"EZ-SmartFarm_BachN/handlers"
)

func SetupRoutes() {
	// Health check endpoint
	http.HandleFunc("/health", handlers.HealthCheck)

	// Coop routes (CRUD operations)
	http.HandleFunc("/api/coops", handleCoops)

	// Egg routes (CRUD operations)
	http.HandleFunc("/api/eggs", handleEggs)

	// Vaccine routes (CRUD operations)
	http.HandleFunc("/api/vaccines", handleVaccines)

	// Health routes (CRUD operations)
	http.HandleFunc("/api/healths", handleHealths)
}

// handleCoops routes requests to appropriate handler based on method
func handleCoops(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodPost:
		// Create new coop
		handlers.CreateCoopHandler(w, r)
	case http.MethodGet:
		if id != "" {
			// Get single coop by ID
			handlers.GetCoopHandler(w, r)
		} else {
			// Get all coops
			handlers.GetAllCoopsHandler(w, r)
		}
	case http.MethodPut:
		// Update coop
		handlers.UpdateCoopHandler(w, r)
	case http.MethodDelete:
		// Delete coop
		handlers.DeleteCoopHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleEggs routes requests to appropriate handler based on method
func handleEggs(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	switch r.Method {
	case http.MethodPost:
		// Create new egg record
		handlers.CreateEggHandler(w, r)
	case http.MethodGet:
		if id != "" {
			// Get single egg record by ID
			handlers.GetEggHandler(w, r)
		} else if coopID != "" {
			// Get all egg records for a specific coop
			handlers.GetEggsByCoopHandler(w, r)
		} else {
			// Get all egg records
			handlers.GetAllEggsHandler(w, r)
		}
	case http.MethodPut:
		// Update egg record
		handlers.UpdateEggHandler(w, r)
	case http.MethodDelete:
		// Delete egg record
		handlers.DeleteEggHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHealths routes requests to appropriate handler based on method
func handleHealths(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	switch r.Method {
	case http.MethodPost:
		handlers.CreateHealthHandler(w, r)
	case http.MethodGet:
		if id != "" {
			handlers.GetHealthHandler(w, r)
		} else if coopID != "" {
			handlers.GetHealthsByCoopHandler(w, r)
		} else {
			handlers.GetAllHealthsHandler(w, r)
		}
	case http.MethodPut:
		handlers.UpdateHealthHandler(w, r)
	case http.MethodDelete:
		handlers.DeleteHealthHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleVaccines routes requests to appropriate handler based on method
func handleVaccines(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	switch r.Method {
	case http.MethodPost:
		handlers.CreateVaccineHandler(w, r)
	case http.MethodGet:
		if id != "" {
			handlers.GetVaccineHandler(w, r)
		} else if coopID != "" {
			handlers.GetVaccinesByCoopHandler(w, r)
		} else {
			handlers.GetAllVaccinesHandler(w, r)
		}
	case http.MethodPut:
		handlers.UpdateVaccineHandler(w, r)
	case http.MethodDelete:
		handlers.DeleteVaccineHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
