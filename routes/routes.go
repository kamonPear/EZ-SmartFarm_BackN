package routes

import (
	"net/http"

	"gorm.io/gorm"

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
	rg.POST("/login", handlers.Login)

	// Coops
	rg.POST("/api/coops", handlers.CreateCoopHandler)
	rg.GET("/api/coops", handleCoopsGet)
	rg.PUT("/api/coops", handlers.UpdateCoopHandler)
	rg.DELETE("/api/coops", handlers.DeleteCoopHandler)

	rg.POST("/api/coops/layout", handlers.SaveCoopLayoutHandler)

	// Eggs
	rg.POST("/api/eggs", handlers.CreateEggHandler)
	rg.GET("/api/eggs", handleEggsGet)
	rg.PUT("/api/eggs", handlers.UpdateEggHandler)
	rg.DELETE("/api/eggs", handlers.DeleteEggHandler)

	// Health checks
	rg.POST("/api/healths", handlers.CreateHealthHandler)
	rg.GET("/api/healths", handleHealthsGet)
	rg.PUT("/api/healths", handlers.UpdateHealthHandler)
	rg.DELETE("/api/healths", handlers.DeleteHealthHandler)
	rg.GET("/api/notifications/health_checks", handlers.GetHealthCheckNotiHandler)

	// Foodstock (current totals; stock is added via /api/importfoods)
	rg.GET("/api/foods", handleFoodsGet)
	rg.PUT("/api/foods", handlers.UpdateFoodstockHandler)
	rg.DELETE("/api/foods", handlers.DeleteFoodstockHandler)
	rg.POST("/api/foodstocks/force-deduct", handlers.ForceDeductStockHandler)

	// Food import lots (each lot adds onto foodstock automatically)
	rg.POST("/api/importfoods", handlers.CreateImportFoodHandler)
	rg.GET("/api/importfoods", handleImportFoodsGet)
	rg.GET("/api/food_history", handlers.GetFoodHistoryHandler)

	// Vaccines / medicine schedule
	// NOTE: register the more specific /api/vaccines/* paths too - Go's
	// ServeMux matches the most specific pattern, so ordering here doesn't matter.
	rg.GET("/api/vaccines/recommended", handlers.GetRecommendedVaccinesHandler)
	rg.POST("/api/vaccines/schedule", handlers.AddCustomMedicineHandler)
	rg.PUT("/api/vaccines/schedule/update", handlers.UpdateCustomMedicineHandler)
	rg.POST("/api/vaccines", handlers.CreateVaccineHandler)
	rg.GET("/api/vaccines", handlers.GetVaccineHandler)
	rg.DELETE("/api/vaccines", handlers.DeleteVaccineHandler)

	// Calendar alerts: GET fetches, PUT toggles completion, DELETE removes a schedule
	rg.GET("/api/vaccines/alerts", handlers.GetVaccineCalendarAlertsHandler)
	rg.PUT("/api/vaccines/alerts", handlers.GetVaccineCalendarAlertsHandler)
	rg.DELETE("/api/vaccines/alerts", handlers.GetVaccineCalendarAlertsHandler)

	rg.GET("/api/notifications/vaccines", handlers.GetVaccineNotificationsHandler)

	// Devices
	rg.POST("/api/devices", handlers.CreateDeviceHandler)
	rg.GET("/api/devices", handleDevicesGet)
	rg.PUT("/api/devices", handlers.UpdateDeviceHandler)
	rg.DELETE("/api/devices", handlers.DeleteDeviceHandler)

	// Sensors
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
