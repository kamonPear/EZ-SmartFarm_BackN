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
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	if err := database.InitDatabase(cfg.Database); err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		// Continue running server even if DB fails for development
	} else {
		// Run migrations if database connected successfully
		if err := database.MigrateModels(database.GetDB()); err != nil {
			log.Printf("Warning: Migration failed: %v", err)
		}
	}

	// Setup routes
	routes.SetupRoutes()

	// Start server
	port := ":" + cfg.ServerPort
	fmt.Printf("Server starting on port %s...\n", cfg.ServerPort)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down server...")
		database.CloseDatabase()
		os.Exit(0)
	}()

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
