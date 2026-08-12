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
		&models.ImportFood{},
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

	// name_coop / device.name are secondary keys added alongside the existing id-based FKs.
	// AutoMigrate doesn't manage these on its own, so add them explicitly and idempotently.
	ensureUniqueIndex(db, "coop", "uq_coop_name_coop", "ALTER TABLE `coop` ADD UNIQUE KEY `uq_coop_name_coop` (`name_coop`)")
	ensureUniqueIndex(db, "device", "uq_device_name", "ALTER TABLE `device` ADD UNIQUE KEY `uq_device_name` (`name`)")

	ensureForeignKey(db, "egg", "fk_name_coop_eggs", "ALTER TABLE `egg` ADD CONSTRAINT `fk_name_coop_eggs` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")
	ensureForeignKey(db, "health", "fk_name_coop_healths", "ALTER TABLE `health` ADD CONSTRAINT `fk_name_coop_healths` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")
	ensureForeignKey(db, "vaccine", "fk_name_coop_vaccines", "ALTER TABLE `vaccine` ADD CONSTRAINT `fk_name_coop_vaccines` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")
	ensureForeignKey(db, "device", "fk_name_coop_devices", "ALTER TABLE `device` ADD CONSTRAINT `fk_name_coop_devices` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")
	ensureForeignKey(db, "sensor_log", "fk_device_name_sensor_logs", "ALTER TABLE `sensor_log` ADD CONSTRAINT `fk_device_name_sensor_logs` FOREIGN KEY (`name`) REFERENCES `device`(`name`)")

	fmt.Println("✓ All tables migrated successfully")
	return nil
}

// ensureUniqueIndex adds a unique index only if it doesn't already exist (AutoMigrate re-runs on every startup)
func ensureUniqueIndex(db *gorm.DB, table, indexName, ddl string) {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?",
		table, indexName,
	).Scan(&count).Error; err != nil {
		log.Printf("Warning: could not check index %s: %v", indexName, err)
		return
	}
	if count > 0 {
		return
	}
	if err := db.Exec(ddl).Error; err != nil {
		log.Printf("Warning: could not add unique index %s: %v", indexName, err)
	}
}

// ensureForeignKey adds a foreign key constraint only if it doesn't already exist (AutoMigrate re-runs on every startup)
func ensureForeignKey(db *gorm.DB, table, constraintName, ddl string) {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = ? AND CONSTRAINT_NAME = ?",
		table, constraintName,
	).Scan(&count).Error; err != nil {
		log.Printf("Warning: could not check constraint %s: %v", constraintName, err)
		return
	}
	if count > 0 {
		return
	}
	if err := db.Exec(ddl).Error; err != nil {
		log.Printf("Warning: could not add foreign key %s: %v", constraintName, err)
	}
}
