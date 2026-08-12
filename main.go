package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"EZ-SmartFarm_BachN/config"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/routes"     
	"EZ-SmartFarm_BachN/scheduler" // 🌟 1. เพิ่ม import scheduler
	"EZ-SmartFarm_BachN/services"
)

func main() {
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