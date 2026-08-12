package routes

import (
	"encoding/json" // 🌟 เพิ่มตัวนี้สำหรับแปลง JSON
	"net/http"

	"gorm.io/gorm"

	"EZ-SmartFarm_BachN/database" // 🌟 เพิ่มตัวนี้เพื่อเรียกฟังก์ชันคำนวณแจ้งเตือน
	"EZ-SmartFarm_BachN/handlers"
)

// SetupRoutes เติม db *gorm.DB ไว้ในวงเล็บ เพื่อรับค่า db มาจาก main.go
func SetupRoutes(db *gorm.DB) {
	// Health check endpoint
	http.HandleFunc("/health", handlers.HealthCheck)

	// Coop routes (CRUD operations)
	http.HandleFunc("/api/coops", handleCoops)

	// เส้นทางสำหรับจัดการข้อมูลไข่
	http.HandleFunc("/api/eggs", handleEggs)

	http.HandleFunc("/api/healths", handleHealths)

	http.HandleFunc("/api/foods", handleFoods)

	// เส้นทางสำหรับบันทึกการนำเข้าอาหารแต่ละล็อต (จะไปบวกเพิ่มใน foodstock อัตโนมัติ)
	http.HandleFunc("/api/importfoods", handleImportFoods)

	// เส้นทางสำหรับประวัติการนำอาหารเข้า
	http.HandleFunc("/api/food_history", handleFoodHistory)

	http.HandleFunc("/api/coops/layout", handleCoopLayout)

	// เส้นทางวัคซีน/ยา
	http.HandleFunc("/api/vaccines/recommended", handleVaccines)
	http.HandleFunc("/api/vaccines/schedule", handleVaccines) 
	http.HandleFunc("/api/vaccines", handleVaccines)

	http.HandleFunc("/api/devices", handleDevices)

	http.HandleFunc("/api/sensor-logs", handlers.ReceiveSensorDataHandler)

	http.HandleFunc("/api/sensor/upload", handlers.HandleArduinoUpload(db))

	http.HandleFunc("/login", handlers.Login)

	// สำหรับหน้าจอตัวปฏิทินแสดงจุดเตือน
	http.HandleFunc("/api/vaccines/alerts", handlers.GetVaccineCalendarAlertsHandler)

	

	http.HandleFunc("/api/notifications/vaccines", handlers.GetVaccineNotificationsHandler)

	http.HandleFunc("/api/foodstocks/force-deduct", handlers.ForceDeductStockHandler)

	http.HandleFunc("/api/notifications/health_checks", handlers.GetHealthCheckNotiHandler)

	http.HandleFunc("/api/vaccines/schedule/update", handlers.UpdateCustomMedicineHandler)
}


// handleCoops routes requests to appropriate handler based on method
func handleCoops(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodPost:
		handlers.CreateCoopHandler(w, r)
	case http.MethodGet:
		if id != "" {
			handlers.GetCoopHandler(w, r)
		} else {
			handlers.GetAllCoopsHandler(w, r)
		}
	case http.MethodPut:
		handlers.UpdateCoopHandler(w, r)
	case http.MethodDelete:
		handlers.DeleteCoopHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// อัปเดตฟังก์ชัน handleEggs ให้รองรับ GET, PUT, DELETE
func handleEggs(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	switch r.Method {
	case http.MethodPost:
		handlers.CreateEggHandler(w, r)
	case http.MethodGet:
		// รองรับการดึงข้อมูลทั้งแบบระบุ ID, ระบุคอก และดึงทั้งหมด
		if id != "" {
			handlers.GetEggHandler(w, r)
		} else if coopID != "" {
			handlers.GetEggsByCoopHandler(w, r)
		} else {
			handlers.GetAllEggsHandler(w, r) 
		}
	case http.MethodPut:
		handlers.UpdateEggHandler(w, r)
	case http.MethodDelete:
		handlers.DeleteEggHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

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

// ฟังก์ชันจัดการคลังอาหาร (ยอดรวมปัจจุบัน - ดู/แก้ไข/ลบเท่านั้น การเพิ่มสต็อกทำผ่าน /api/importfoods)
func handleFoods(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodGet:
		if id != "" {
			handlers.GetFoodstockHandler(w, r)
		} else {
			handlers.GetAllFoodstocksHandler(w, r)
		}
	case http.MethodPut:
		handlers.UpdateFoodstockHandler(w, r)
	case http.MethodDelete:
		handlers.DeleteFoodstockHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ฟังก์ชันจัดการล็อตการนำเข้าอาหาร (สร้างล็อตใหม่จะไปบวกเพิ่มใน foodstock อัตโนมัติ)
func handleImportFoods(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	switch r.Method {
	case http.MethodPost:
		handlers.CreateImportFoodHandler(w, r)
	case http.MethodGet:
		if id != "" {
			handlers.GetImportFoodHandler(w, r)
		} else {
			handlers.GetAllImportFoodsHandler(w, r)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ฟังก์ชันจัดการ Layout และอนุญาต CORS
func handleCoopLayout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPost {
		handlers.SaveCoopLayoutHandler(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ปรับปรุง: การคัดแยกและตรวจสอบเส้นทางย่อยภายใต้บริการข้อมูลวัคซีน/ตัวยาหลัก
func handleVaccines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path == "/api/vaccines/recommended" {
		if r.Method == http.MethodGet {
			handlers.GetRecommendedVaccinesHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else if r.URL.Path == "/api/vaccines/schedule" {
		if r.Method == http.MethodPost {
			handlers.AddCustomMedicineHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateVaccineHandler(w, r)
		case http.MethodGet:
			handlers.GetVaccineHandler(w, r)
		case http.MethodDelete:
			handlers.DeleteVaccineHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ฟังก์ชันจัดการข้อมูลอุปกรณ์ (Device)
func handleDevices(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	coopID := r.URL.Query().Get("coop_id")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodPost:
		handlers.CreateDeviceHandler(w, r)
	case http.MethodGet:
		if id != "" {
			handlers.GetDeviceHandler(w, r)
		} else if coopID != "" {
			handlers.GetDevicesByCoopHandler(w, r)
		} else {
			handlers.GetAllDevicesHandler(w, r) 
		}
	case http.MethodPut:
		handlers.UpdateDeviceHandler(w, r)
	case http.MethodDelete:
		handlers.DeleteDeviceHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// เพิ่มฟังก์ชันจัดการประวัติอาหาร
func handleFoodHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handlers.GetFoodHistoryHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ==========================================
// 🌟 ส่วนที่เพิ่มใหม่ สำหรับการดึงข้อมูลแจ้งเตือน
// ==========================================
func handleVaccineNotifications(w http.ResponseWriter, r *http.Request) {
	// อนุญาต CORS เพื่อให้แอป Flutter หรือเบราว์เซอร์เข้าถึงได้
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		// เรียกใช้ฟังก์ชันคำนวณอายุไก่เทียบกับเกณฑ์
		alerts, err := database.GetPendingVaccinesAlerts()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// แปลงข้อมูลที่ได้เป็น JSON แล้วส่งกลับไป
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(alerts)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}