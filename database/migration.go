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

	// name_coop is a secondary key added alongside the existing id-based FKs.
	// AutoMigrate doesn't manage this on its own, so add it explicitly and idempotently.
	ensureUniqueIndex(db, "coop", "uq_coop_name_coop", "ALTER TABLE `coop` ADD UNIQUE KEY `uq_coop_name_coop` (`name_coop`)")

	ensureForeignKey(db, "egg", "fk_name_coop_eggs", "ALTER TABLE `egg` ADD CONSTRAINT `fk_name_coop_eggs` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")
	ensureForeignKey(db, "health", "fk_name_coop_healths", "ALTER TABLE `health` ADD CONSTRAINT `fk_name_coop_healths` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")
	ensureForeignKey(db, "vaccine", "fk_name_coop_vaccines", "ALTER TABLE `vaccine` ADD CONSTRAINT `fk_name_coop_vaccines` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")
	ensureForeignKey(db, "device", "fk_name_coop_devices", "ALTER TABLE `device` ADD CONSTRAINT `fk_name_coop_devices` FOREIGN KEY (`name_coop`) REFERENCES `coop`(`name_coop`)")

	// device.name used to be unique so it could double as a natural key for sensor_log.
	// That blocked placing more than one sensor of the same type in a coop, so both the FK
	// and the unique index are dropped here (idempotent - only acts if still present from
	// an earlier deploy). (coop_id, slot_index) is the real identity for a placed device now.
	ensureForeignKeyDropped(db, "sensor_log", "fk_device_name_sensor_logs")
	ensureIndexDropped(db, "device", "uq_device_name")

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

// ensureIndexDropped removes an index if it's still present (used to walk back constraints added by earlier migrations)
func ensureIndexDropped(db *gorm.DB, table, indexName string) {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?",
		table, indexName,
	).Scan(&count).Error; err != nil {
		log.Printf("Warning: could not check index %s: %v", indexName, err)
		return
	}
	if count == 0 {
		return
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", table, indexName)).Error; err != nil {
		log.Printf("Warning: could not drop index %s: %v", indexName, err)
	}
}

// ensureForeignKeyDropped removes a foreign key constraint if it's still present
func ensureForeignKeyDropped(db *gorm.DB, table, constraintName string) {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = ? AND CONSTRAINT_NAME = ?",
		table, constraintName,
	).Scan(&count).Error; err != nil {
		log.Printf("Warning: could not check constraint %s: %v", constraintName, err)
		return
	}
	if count == 0 {
		return
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", table, constraintName)).Error; err != nil {
		log.Printf("Warning: could not drop foreign key %s: %v", constraintName, err)
	}
}
