// =====================================================================================
// ไฟล์ routes/routes.go - "แผนที่" ของ API: URL ไหนไปเรียก handler ตัวไหน
// =====================================================================================
//
// เมื่อหน้าบ้านเรียก เช่น GET /api/coops ระบบจะดูตารางในไฟล์นี้ว่าต้องส่งต่อให้ใคร
// แต่ละบรรทัดอ่านได้เป็น: เมทอด + path + (ยาม) + handler
//
// ยามที่ครอบ (มาจาก auth/middleware.go):
//  - ไม่มียาม                     -> เปิดสาธารณะ: /health, login, รับค่าจากเซนเซอร์ (บอร์ด ESP32 ไม่มี JWT)
//  - auth.RequireAuth            -> ต้อง login (แนบ JWT) ใช้กับเกือบทุก API
//  - auth.RequireAdminKey        -> ต้องมีรหัสผู้ดูแล: สร้าง/ลบบัญชี
//  - auth.RequireScheduledTaskKey -> ต้องมีรหัสงานตั้งเวลา: ตัดสต็อกอาหารอัตโนมัติ
//
// ใช้ http.ServeMux ของ Go 1.22 ขึ้นไป ที่รองรับรูปแบบ "GET /path" (ระบุเมทอดในตัว) ได้เลย
// จึงไม่ต้องพึ่งเฟรมเวิร์กภายนอกอย่าง Gin/Echo
package routes

import (
	"net/http"

	"gorm.io/gorm"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/handlers"
)

// router ห่อ http.ServeMux ไว้ เพื่อให้เขียนลงทะเบียนเส้นทางสั้น ๆ อ่านง่าย เช่น rg.GET(path, handler)
// แทนการเขียน mux.HandleFunc("GET "+path, handler) ยาว ๆ ทุกบรรทัด
type router struct {
	mux *http.ServeMux
}

// GET ลงทะเบียน handler ให้ path นี้ เฉพาะคำขอแบบ GET (ดึงข้อมูล)
// POST = เพิ่มข้อมูล, PUT = แก้ไขข้อมูล, DELETE = ลบข้อมูล (3 ฟังก์ชันถัดไปทำงานแบบเดียวกัน)
func (rg *router) GET(path string, h http.HandlerFunc)    { rg.mux.HandleFunc("GET "+path, h) }
func (rg *router) POST(path string, h http.HandlerFunc)   { rg.mux.HandleFunc("POST "+path, h) }
func (rg *router) PUT(path string, h http.HandlerFunc)    { rg.mux.HandleFunc("PUT "+path, h) }
func (rg *router) DELETE(path string, h http.HandlerFunc) { rg.mux.HandleFunc("DELETE "+path, h) }

// ANY ลงทะเบียน handler ให้ path นี้ทุกเมทอด (ให้ handler ตัดสินใจเองว่ารับอะไร)
// ใช้กับ /health ที่ Render เรียกเช็กว่าเซิร์ฟเวอร์ยังทำงานอยู่
func (rg *router) ANY(path string, h http.HandlerFunc) { rg.mux.HandleFunc(path, h) }

// SetupRoutes ลงทะเบียน API ทุกเส้นของระบบ ถูกเรียกครั้งเดียวจาก main.go
//
// รับ db *gorm.DB มาจาก main.go เพื่อส่งต่อให้ handler ที่ต้องใช้ db ตรง ๆ
// (ตอนนี้คือ HandleArduinoUpload ที่รับข้อมูลจากบอร์ด) handler อื่นใช้ database.GetDB() เอง
//
// หมวด API หลัก:
//  - Auth          : login / register / ลบผู้ใช้ / ดูข้อมูลตัวเอง
//  - Coops         : คอกไก่ + ตำแหน่งบนผังฟาร์ม, farm-layout (รูปทรงผัง)
//  - Farm threshold: ค่ามาตรฐานอุณหภูมิ/แอมโมเนีย (ย้ายมาเก็บใน DB เว็บกับแอปจึงเห็นค่าเดียวกัน)
//  - Eggs / Healths: บันทึกไข่, ผลตรวจสุขภาพ
//  - Foods         : คลังอาหาร, ตัดสต็อก, ประวัติแจกจ่าย, นำเข้าอาหาร (importfoods)
//  - Data purge    : ลบข้อมูลที่สำรองแล้ว
//  - Vaccines      : ประเภทวัคซีน, ตาราง, แจ้งเตือนในปฏิทิน
//  - Health appointments / Calendar markers: นัดตรวจ และจุดบนปฏิทินรวม
//  - Devices / Device types: อุปกรณ์ในคอก และชนิดอุปกรณ์ที่ผู้ใช้สร้างเอง
//  - Sensors       : รับค่าจากบอร์ด (สาธารณะ) และดูการแจ้งเตือนความเคลื่อนไหว
func SetupRoutes(db *gorm.DB) {
	rg := &router{mux: http.DefaultServeMux}

	rg.ANY("/health", handlers.HealthCheck)

	// Auth
	rg.POST("/api/auth/login", handlers.LoginHandler)                              // public
	rg.POST("/api/auth/register", auth.RequireAdminKey(handlers.RegisterHandler))  // needs X-Admin-Key
	rg.DELETE("/api/auth/users", auth.RequireAdminKey(handlers.DeleteUserHandler)) // needs X-Admin-Key
	rg.GET("/api/auth/me", auth.RequireAuth(handlers.MeHandler))                   // any logged-in user

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

	// Farm threshold (target temperature/ammonia shared across every coop) - was
	// localStorage/SharedPreferences-only on each platform before (web and app
	// showed different values); now backed by the DB so both stay in sync
	rg.GET("/api/farm-threshold", auth.RequireAuth(handlers.GetFarmThresholdHandler))
	rg.PUT("/api/farm-threshold", auth.RequireAuth(handlers.UpdateFarmThresholdHandler))
	rg.GET("/api/food-age-rules", auth.RequireAuth(handlers.GetFoodAgeRulesHandler))
	rg.PUT("/api/food-age-rules", auth.RequireAuth(handlers.UpdateFoodAgeRulesHandler))

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
	rg.PUT("/api/foods/min-quantity", auth.RequireAuth(handlers.UpdateFoodMinQuantityHandler))
	rg.POST("/api/foodstocks/force-deduct", auth.RequireAuth(handlers.ForceDeductStockHandler))
	// ปลุกจากภายนอก (เช่น cron-job.org) แทน JWT - ดู auth.RequireScheduledTaskKey
	rg.POST("/api/system/food-deduction/run", auth.RequireScheduledTaskKey(handlers.RunScheduledFoodDeductionHandler))
	rg.GET("/api/foods/coop-consumption", auth.RequireAuth(handlers.GetCoopFoodConsumptionHandler))
	rg.POST("/api/foods/distribution", auth.RequireAuth(handlers.RecordFoodDistributionHandler))
	rg.GET("/api/foods/distribution", auth.RequireAuth(handlers.GetFoodDistributionHistoryHandler))
	rg.DELETE("/api/foods/distribution", auth.RequireAuth(handlers.DeleteAllFoodDistributionHandler))

	// Delete the caller's own records by category (used after "backup" -> "delete this data")
	rg.POST("/api/data/purge", auth.RequireAuth(handlers.PurgeDataHandler))

	// Food import lots (each lot adds onto foodstock automatically)
	rg.POST("/api/importfoods", auth.RequireAuth(handlers.CreateImportFoodHandler))
	rg.GET("/api/importfoods", auth.RequireAuth(handleImportFoodsGet))
	rg.DELETE("/api/importfoods", auth.RequireAuth(handlers.DeleteAllImportFoodsHandler))
	rg.GET("/api/food_history", auth.RequireAuth(handlers.GetFoodHistoryHandler))

	// Vaccines / medicine schedule
	// NOTE: register the more specific /api/vaccines/* paths too - Go's
	// ServeMux matches the most specific pattern, so ordering here doesn't matter.
	rg.GET("/api/vaccines/recommended", auth.RequireAuth(handlers.GetRecommendedVaccinesHandler))
	rg.GET("/api/vaccines/schedule/matching-coops", auth.RequireAuth(handlers.GetVaccineMatchingCoopsHandler))
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

	// Manual health-check appointments ("นัดตรวจสุขภาพเอง") - booked ahead of time,
	// independent of the auto vaccine-based schedule; folded into /api/calendar/markers
	rg.POST("/api/health-appointments", auth.RequireAuth(handlers.CreateHealthAppointmentHandler))
	rg.GET("/api/health-appointments", auth.RequireAuth(handleHealthAppointmentsGet))
	rg.DELETE("/api/health-appointments", auth.RequireAuth(handlers.DeleteHealthAppointmentHandler))

	// Unified calendar markers - single source of truth for "what's on this day"
	// across vaccines/health/coop birthdays/manual appointments, shared by both the
	// web app and the Flutter app so neither has to re-derive it client-side
	rg.GET("/api/calendar/markers", auth.RequireAuth(handlers.GetCalendarMarkersHandler))

	// Devices
	rg.POST("/api/devices", auth.RequireAuth(handlers.CreateDeviceHandler))
	rg.GET("/api/devices", auth.RequireAuth(handleDevicesGet))
	rg.PUT("/api/devices", auth.RequireAuth(handlers.UpdateDeviceHandler))
	rg.DELETE("/api/devices", auth.RequireAuth(handlers.DeleteDeviceHandler))

	// Device types - ชนิดอุปกรณ์ที่ผู้ใช้เพิ่มเอง (ชื่อ+ไอคอน) ไว้เลือกใช้ซ้ำตอน
	// จัดวางอุปกรณ์ลงคอก แยกจาก Device ซึ่งเป็นอุปกรณ์ที่ถูกวางจริงแล้ว
	rg.POST("/api/device-types", auth.RequireAuth(handlers.CreateDeviceTypeHandler))
	rg.GET("/api/device-types", auth.RequireAuth(handlers.GetDeviceTypesHandler))
	rg.PUT("/api/device-types", auth.RequireAuth(handlers.UpdateDeviceTypeHandler))
	rg.DELETE("/api/device-types", auth.RequireAuth(handlers.DeleteDeviceTypeHandler))

	// Sensors - IoT device ingestion, no user JWT available in the field, stays public
	rg.POST("/api/sensor-logs", handlers.ReceiveSensorDataHandler)
	rg.POST("/api/sensor/upload", handlers.HandleArduinoUpload(db))
	rg.GET("/api/motion-alerts", auth.RequireAuth(handlers.GetMotionAlertsHandler))
}

// handleCoopsGet แยกเส้นทาง GET /api/coops เป็น 2 กรณีตาม query:
//  - มี ?id=  -> GetCoopHandler (ดูคอกเดียว)
//  - ไม่มี    -> GetAllCoopsHandler (ดูทุกคอกของผู้ใช้)
// ทำไมต้องแยกแบบนี้: ServeMux จับคู่จาก path อย่างเดียว ไม่ได้ดู query string
func handleCoopsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") != "" {
		handlers.GetCoopHandler(w, r)
	} else {
		handlers.GetAllCoopsHandler(w, r)
	}
}

// handleEggsGet แยก GET /api/eggs ตาม query:
//  - ?id=       -> ดูรายการไข่รายการเดียว
//  - ?coop_id=  -> ดูไข่ทั้งหมดของคอกนั้น
//  - ไม่มี       -> ดูไข่ทั้งหมดของผู้ใช้
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

// handleHealthsGet แยก GET /api/healths ตาม query: ?id= (รายการเดียว) / ?coop_id= (ทั้งคอก) / ไม่มี (ทั้งหมด)
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

// handleHealthAppointmentsGet แยก GET /api/health-appointments: ?coop_id= (นัดของคอกนั้น) / ไม่มี (นัดทั้งหมด)
func handleHealthAppointmentsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("coop_id") != "" {
		handlers.GetHealthAppointmentsByCoopHandler(w, r)
	} else {
		handlers.GetAllHealthAppointmentsHandler(w, r)
	}
}

// handleFoodsGet แยก GET /api/foods: ?id= (สต็อกอาหารประเภทเดียว) / ไม่มี (สต็อกทุกประเภท)
func handleFoodsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") != "" {
		handlers.GetFoodstockHandler(w, r)
	} else {
		handlers.GetAllFoodstocksHandler(w, r)
	}
}

// handleImportFoodsGet แยก GET /api/importfoods: ?id= (ล็อตนำเข้าเดียว) / ไม่มี (ทุกล็อต)
func handleImportFoodsGet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("id") != "" {
		handlers.GetImportFoodHandler(w, r)
	} else {
		handlers.GetAllImportFoodsHandler(w, r)
	}
}

// handleDevicesGet แยก GET /api/devices: ?id= (อุปกรณ์ตัวเดียว) / ?coop_id= (อุปกรณ์ในคอกนั้น) / ไม่มี (ทั้งหมด)
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
