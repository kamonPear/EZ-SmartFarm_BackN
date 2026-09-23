package routes

import (
	"net/http"

	"gorm.io/gorm"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/handlers"
)

// router is a thin wrapper around http.ServeMux that registers one handler per
// HTTP method, so every route below reads as method + path + handler at a glance.
type router struct {
	mux *http.ServeMux
}

func (rg *router) GET(path string, h http.HandlerFunc)    { rg.mux.HandleFunc("GET "+path, h) }
func (rg *router) POST(path string, h http.HandlerFunc)   { rg.mux.HandleFunc("POST "+path, h) }
func (rg *router) PUT(path string, h http.HandlerFunc)    { rg.mux.HandleFunc("PUT "+path, h) }
func (rg *router) DELETE(path string, h http.HandlerFunc) { rg.mux.HandleFunc("DELETE "+path, h) }

// ANY registers a handler for a path regardless of method (the handler itself
// decides what it accepts, or accepts everything).
func (rg *router) ANY(path string, h http.HandlerFunc) { rg.mux.HandleFunc(path, h) }

// SetupRoutes เติม db *gorm.DB ไว้ในวงเล็บ เพื่อรับค่า db มาจาก main.go
func SetupRoutes(db *gorm.DB) {
	rg := &router{mux: http.DefaultServeMux}

	rg.ANY("/health", handlers.HealthCheck)

	// Auth
	rg.POST("/api/auth/login", handlers.LoginHandler)                      // public
	rg.POST("/api/auth/register", auth.RequireAdminKey(handlers.RegisterHandler)) // needs X-Admin-Key
	rg.GET("/api/auth/me", auth.RequireAuth(handlers.MeHandler))               // any logged-in user

	// Coops
	rg.POST("/api/coops", auth.RequireAuth(handlers.CreateCoopHandler))
	rg.GET("/api/coops", auth.RequireAuth(handleCoopsGet))
	rg.PUT("/api/coops", auth.RequireAuth(handlers.UpdateCoopHandler))
	rg.DELETE("/api/coops", auth.RequireAuth(handlers.DeleteCoopHandler))

	rg.POST("/api/coops/layout", auth.RequireAuth(handlers.SaveCoopLayoutHandler))
	rg.PUT("/api/coops/positions", auth.RequireAuth(handlers.UpdateCoopPositionsHandler))

	// Farm layout (chosen outline shape coops are arranged within)
	rg.GET("/api/farm-layout", auth.RequireAuth(handlers.GetFarmLayoutHandler))
	rg.PUT("/api/farm-layout", auth.RequireAuth(handlers.UpdateFarmLayoutHandler))

	// Eggs
	rg.POST("/api/eggs", auth.RequireAuth(handlers.CreateEggHandler))
	rg.GET("/api/eggs", auth.RequireAuth(handleEggsGet))
	rg.PUT("/api/eggs", auth.RequireAuth(handlers.UpdateEggHandler))
	rg.DELETE("/api/eggs", auth.RequireAuth(handlers.DeleteEggHandler))

	// Health checks
	rg.POST("/api/healths", auth.RequireAuth(handlers.CreateHealthHandler))
	rg.GET("/api/healths", auth.RequireAuth(handleHealthsGet))
	rg.PUT("/api/healths", auth.RequireAuth(handlers.UpdateHealthHandler))
	rg.DELETE("/api/healths", auth.RequireAuth(handlers.DeleteHealthHandler))
	rg.GET("/api/notifications/health_checks", auth.RequireAuth(handlers.GetHealthCheckNotiHandler))

	// Foodstock (current totals; stock is added via /api/importfoods)
	rg.GET("/api/foods", auth.RequireAuth(handleFoodsGet))
	rg.PUT("/api/foods", auth.RequireAuth(handlers.UpdateFoodstockHandler))
	rg.DELETE("/api/foods", auth.RequireAuth(handlers.DeleteFoodstockHandler))
	rg.POST("/api/foodstocks/force-deduct", auth.RequireAuth(handlers.ForceDeductStockHandler))
	rg.GET("/api/foods/coop-consumption", auth.RequireAuth(handlers.GetCoopFoodConsumptionHandler))
	rg.POST("/api/foods/distribution", auth.RequireAuth(handlers.RecordFoodDistributionHandler))
	rg.GET("/api/foods/distribution", auth.RequireAuth(handlers.GetFoodDistributionHistoryHandler))
	rg.DELETE("/api/foods/distribution", auth.RequireAuth(handlers.DeleteAllFoodDistributionHandler))

	// Food import lots (each lot adds onto foodstock automatically)
	rg.POST("/api/importfoods", auth.RequireAuth(handlers.CreateImportFoodHandler))
	rg.GET("/api/importfoods", auth.RequireAuth(handleImportFoodsGet))
	rg.DELETE("/api/importfoods", auth.RequireAuth(handlers.DeleteAllImportFoodsHandler))
	rg.GET("/api/food_history", auth.RequireAuth(handlers.GetFoodHistoryHandler))

	// Vaccines / medicine schedule
	// NOTE: register the more specific /api/vaccines/* paths too - Go's
	// ServeMux matches the most specific pattern, so ordering here doesn't matter.
	rg.GET("/api/vaccines/recommended", auth.RequireAuth(handlers.GetRecommendedVaccinesHandler))
	rg.POST("/api/vaccines/schedule", auth.RequireAuth(handlers.AddCustomMedicineHandler))
	rg.GET("/api/vaccines/schedule", auth.RequireAuth(handlers.GetMedicineSchedulesHandler))
	rg.PUT("/api/vaccines/schedule/update", auth.RequireAuth(handlers.UpdateCustomMedicineHandler))
	rg.POST("/api/vaccines", auth.RequireAuth(handlers.CreateVaccineHandler))
	rg.GET("/api/vaccines", auth.RequireAuth(handlers.GetVaccineHandler))
	rg.PUT("/api/vaccines", auth.RequireAuth(handlers.UpdateVaccineHandler))
	rg.DELETE("/api/vaccines", auth.RequireAuth(handlers.DeleteVaccineHandler))

	// Calendar alerts: GET fetches, PUT toggles completion, DELETE removes a schedule
	rg.GET("/api/vaccines/alerts", auth.RequireAuth(handlers.GetVaccineCalendarAlertsHandler))
	rg.PUT("/api/vaccines/alerts", auth.RequireAuth(handlers.GetVaccineCalendarAlertsHandler))
	rg.DELETE("/api/vaccines/alerts", auth.RequireAuth(handlers.GetVaccineCalendarAlertsHandler))

	rg.GET("/api/notifications/vaccines", auth.RequireAuth(handlers.GetVaccineNotificationsHandler))

	// Devices
	rg.POST("/api/devices", auth.RequireAuth(handlers.CreateDeviceHandler))
	rg.GET("/api/devices", auth.RequireAuth(handleDevicesGet))
	rg.PUT("/api/devices", auth.RequireAuth(handlers.UpdateDeviceHandler))
	rg.DELETE("/api/devices", auth.RequireAuth(handlers.DeleteDeviceHandler))

	// Sensors - IoT device ingestion, no user JWT available in the field, stays public
	rg.POST("/api/sensor-logs", handlers.ReceiveSensorDataHandler)
	rg.POST("/api/sensor/upload", handlers.HandleArduinoUpload(db))
}

// handleCoopsGet dispatches GET /api/coops to a single-record or list lookup
// depending on whether an id query parameter is present.
func handleCoopsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") != "" {
		handlers.GetCoopHandler(w, r)
	} else {
		handlers.GetAllCoopsHandler(w, r)
	}
}

func handleEggsGet(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Query().Get("id") != "":
		handlers.GetEggHandler(w, r)
	case r.URL.Query().Get("coop_id") != "":
		handlers.GetEggsByCoopHandler(w, r)
	default:
		handlers.GetAllEggsHandler(w, r)
	}
}

func handleHealthsGet(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Query().Get("id") != "":
		handlers.GetHealthHandler(w, r)
	case r.URL.Query().Get("coop_id") != "":
		handlers.GetHealthsByCoopHandler(w, r)
	default:
		handlers.GetAllHealthsHandler(w, r)
	}
}

func handleFoodsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") != "" {
		handlers.GetFoodstockHandler(w, r)
	} else {
		handlers.GetAllFoodstocksHandler(w, r)
	}
}

func handleImportFoodsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") != "" {
		handlers.GetImportFoodHandler(w, r)
	} else {
		handlers.GetAllImportFoodsHandler(w, r)
	}
}

func handleDevicesGet(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Query().Get("id") != "":
		handlers.GetDeviceHandler(w, r)
	case r.URL.Query().Get("coop_id") != "":
		handlers.GetDevicesByCoopHandler(w, r)
	default:
		handlers.GetAllDevicesHandler(w, r)
	}
}
