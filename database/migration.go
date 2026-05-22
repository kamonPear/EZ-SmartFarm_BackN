package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// MigrateModels creates all tables in the database
func MigrateModels(db *gorm.DB) error {
	// Disable foreign key checks temporarily to avoid constraint conflicts during migration
	if err := db.Exec("SET FOREIGN_KEY_CHECKS=0").Error; err != nil {
		log.Printf("Warning: Could not disable foreign key checks: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Coop{},
		&models.Device{},
		&models.SensorLog{},
		&models.Egg{},
		&models.Foodstock{},
		&models.Health{},
		&models.Vaccine{},
	); err != nil {
		log.Printf("Migration error: %v", err)
		// Re-enable foreign key checks even if migration fails
		db.Exec("SET FOREIGN_KEY_CHECKS=1")
		return err
	}

	// Re-enable foreign key checks
	if err := db.Exec("SET FOREIGN_KEY_CHECKS=1").Error; err != nil {
		log.Printf("Warning: Could not re-enable foreign key checks: %v", err)
	}

	fmt.Println("✓ All tables migrated successfully")
	return nil
}
