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

	// Foodstock routes (CRUD operations)
	http.HandleFunc("/api/foodstocks", handleFoodstocks)
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

// handleFoodstocks routes requests to appropriate handler based on method
func handleFoodstocks(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodPost:
		// Create new foodstock
		handlers.CreateFoodstockHandler(w, r)
	case http.MethodGet:
		if id != "" {
			// Get single foodstock by ID
			handlers.GetFoodstockHandler(w, r)
		} else {
			// Get all foodstocks
			handlers.GetAllFoodstocksHandler(w, r)
		}
	case http.MethodPut:
		// Update foodstock
		handlers.UpdateFoodstockHandler(w, r)
	case http.MethodDelete:
		// Delete foodstock
		handlers.DeleteFoodstockHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
	

