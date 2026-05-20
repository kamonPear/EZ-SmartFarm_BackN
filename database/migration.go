package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// MigrateModels creates all tables in the database
func MigrateModels(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Coop{},
		&models.Device{},
		&models.SensorLog{},
		&models.Egg{},
		&models.Foodstock{},
		&models.Health{},
		&models.Vaccine{},
		&models.User{},
	); err != nil {
		log.Printf("Migration error: %v", err)
		return err
	}

	fmt.Println("✓ All tables migrated successfully")
	return nil
}
