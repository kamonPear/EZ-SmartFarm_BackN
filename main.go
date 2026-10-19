// =====================================================================================
// ไฟล์ main.go - จุดเริ่มต้นของโปรแกรม backend ทั้งหมด (Go เริ่มรันที่ฟังก์ชัน main เสมอ)
// =====================================================================================
//
// หน้าที่ของไฟล์นี้คือ "เปิดร้าน" ตามลำดับ:
//  1. โหลดค่าตั้งค่า (config) จากไฟล์ .env / environment ของ Render
//  2. ต่อฐานข้อมูล MySQL แล้วสร้าง/อัปเดตตาราง (migration)
//  3. ลงทะเบียน URL ทั้งหมดของ API (routes)
//  4. เปิดงานเบื้องหลัง: MQTT (คุยกับบอร์ด ESP32), ตัวเช็กอุปกรณ์ออฟไลน์, ตัวตัดสต็อกอาหารอัตโนมัติ
//  5. เปิดเว็บเซิร์ฟเวอร์รอรับคำขอจากเว็บ Angular / แอป Flutter / บอร์ด ESP32
//
// โฟลเดอร์ที่ถูกเรียกใช้จากไฟล์นี้:
//  config/    -> อ่านค่าตั้งค่า
//  database/  -> ต่อ DB + migration
//  routes/    -> แผนที่ URL -> handler
//  services/  -> MQTT และตัวเช็กออฟไลน์
//  scheduler/ -> งานตั้งเวลา (cron) ตัดสต็อกอาหาร 18:00 น.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"EZ-SmartFarm_BachN/config"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
	"EZ-SmartFarm_BachN/routes"
	"EZ-SmartFarm_BachN/scheduler" // 🌟 1. เพิ่ม import scheduler
	"EZ-SmartFarm_BachN/services"
)

// main คือฟังก์ชันแรกที่ Go รันเมื่อสั่ง go run . หรือเปิดไฟล์ที่ build แล้ว
//
// ขั้นตอน:
//  0. ตั้งเวลาของทั้งโปรแกรมเป็นเวลาไทย (UTC+7) ก่อนทำอย่างอื่น
//     เครื่องบน Render ใช้เวลา UTC ถ้าไม่ตั้ง time.Now() และเวลาที่เขียนลงฐานข้อมูล
//     (created_at, last_update ฯลฯ) จะช้ากว่าเวลาไทย 7 ชั่วโมง
//     ต้องทำก่อนต่อฐานข้อมูล เพราะการเชื่อมต่อ (loc=Local) จำเขตเวลาไว้ตอนเปิด
//  1. config.LoadConfig() อ่านค่า DB_HOST, DB_USER, SERVER_PORT ฯลฯ
//  2. database.InitDatabase(...) ต่อ MySQL - ถ้าต่อไม่ได้แค่เขียนเตือน ไม่ปิดโปรแกรม
//     (เพื่อให้ /health ยังตอบได้ และ Render ไม่รีสตาร์ทวนไม่จบ)
//  3. database.MigrateModels(...) สร้างตารางที่ยังไม่มี / เพิ่มคอลัมน์ใหม่ (database/migration.go)
//  4. routes.SetupRoutes(db) ลงทะเบียน API ทุกเส้น (routes/routes.go)
//  5. go services.StartMQTTWorker() / go services.StartOfflineChecker()
//     คำว่า go ข้างหน้า = รันเป็น goroutine (งานคู่ขนาน) จะได้ไม่บล็อกการเปิดเซิร์ฟเวอร์
//  6. scheduler.SetupJobs() ตั้ง cron ตัดสต็อกอาหารทุกวัน 18:00 น. (scheduler/cron.go)
//  7. goroutine รอสัญญาณ Ctrl+C / SIGTERM (ตอน Render สั่งปิด) แล้วปิด DB ให้เรียบร้อยก่อนออก
//  8. http.ListenAndServe เปิดเซิร์ฟเวอร์ที่พอร์ตที่ตั้งไว้ โดยครอบทุก request ด้วย enableCORS
func main() {
	time.Local = models.BangkokLocation()

	cfg := config.LoadConfig()

	if err := database.InitDatabase(cfg.Database); err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
	} else {
		if err := database.MigrateModels(database.GetDB()); err != nil {
			log.Printf("Warning: Migration failed: %v", err)
		}
	}

	db := database.GetDB()

	// จัดการ Routes ทั้งหมด (รวมถึง route สำหรับกดตัดสต็อกด้วยมือ)
	routes.SetupRoutes(db)

	// =========================================
	// เริ่มการทำงานของ Background Tasks
	// =========================================
	go services.StartMQTTWorker()
	go services.StartOfflineChecker()

	// 🌟 2. เริ่มการทำงานของ Cron Job สำหรับตัดสต็อกอัตโนมัติ
	scheduler.SetupJobs()
	// =========================================

	port := ":" + cfg.ServerPort
	fmt.Printf("Server starting on port %s...\n", cfg.ServerPort)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down server...")
		database.CloseDatabase()
		os.Exit(0)
	}()

	if err := http.ListenAndServe(port, enableCORS(http.DefaultServeMux)); err != nil {
		fmt.Println("Server error:", err)
	}
}

// enableCORS = "ด่านหน้า" ที่ครอบทุก request ก่อนถึง handler จริง (middleware)
//
// ทำไมต้องมี: เว็บ Angular รันอยู่คนละโดเมนกับ backend (เช่น vercel.app กับ onrender.com)
// เบราว์เซอร์จะบล็อกการเรียกข้ามโดเมนถ้า server ไม่ส่ง header CORS กลับมาอนุญาต
//
// ทำงานอย่างไร:
//  - ใส่ header Access-Control-Allow-* ให้ทุกคำตอบ อนุญาตทุกโดเมน (*), เมทอด GET/POST/PUT/DELETE
//    และ header Content-Type กับ Authorization (ที่ใช้แนบ JWT)
//  - ถ้าเป็นคำขอแบบ OPTIONS (preflight ที่เบราว์เซอร์ถามก่อนส่งจริง) ตอบ 200 ทันทีแล้วจบ
//  - คำขออื่นส่งต่อให้ next (ตัว router) ทำงานตามปกติ
//
// แอป Flutter ไม่ได้รันในเบราว์เซอร์ จึงไม่ติดเรื่อง CORS แต่ไม่มีผลเสียอะไร
// ถูกเรียกจาก: main() ตอน http.ListenAndServe
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
