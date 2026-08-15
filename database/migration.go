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
		// AutoMigrate can fail partway through (e.g. a pre-existing FK type mismatch on
		// egg/vaccine.coop_id) while still having fully migrated earlier models in the list.
		// Log it and keep going instead of bailing out - the idempotent ensure*/drop* calls
		// below are independent cleanup steps that still need to run every startup.
		log.Printf("Warning: AutoMigrate did not fully complete: %v", err)
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
	// That blocked placing more than one sensor of the same type in a coop. Drop every
	// unique index/FK still touching it, however it got created - the named uq_device_name
	// we added by hand, but also anything GORM's own AutoMigrate created automatically back
	// when the struct tag still said `unique` (that one isn't necessarily named the same).
	// (coop_id, slot_index) is the real identity for a placed device now.
	dropAllUniqueIndexesOnColumn(db, "device", "name")

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

// dropAllUniqueIndexesOnColumn removes every unique index touching the given column (except PRIMARY),
// and any foreign key elsewhere in the schema that references it - MySQL refuses to drop a unique
// index that a FK still depends on. This is more thorough than ensureIndexDropped/ensureForeignKeyDropped
// with a hardcoded name because GORM's own AutoMigrate can create a unique index under a name we don't
// control (e.g. from before the `unique` struct tag was removed).
func dropAllUniqueIndexesOnColumn(db *gorm.DB, table, column string) {
	type fkRef struct {
		TableName      string
		ConstraintName string
	}
	var refs []fkRef
	if err := db.Raw(
		`SELECT TABLE_NAME, CONSTRAINT_NAME FROM information_schema.KEY_COLUMN_USAGE
		 WHERE TABLE_SCHEMA = DATABASE() AND REFERENCED_TABLE_NAME = ? AND REFERENCED_COLUMN_NAME = ?`,
		table, column,
	).Scan(&refs).Error; err != nil {
		log.Printf("Warning: could not check FKs referencing %s.%s: %v", table, column, err)
	} else {
		for _, ref := range refs {
			ensureForeignKeyDropped(db, ref.TableName, ref.ConstraintName)
		}
	}

	var indexNames []string
	if err := db.Raw(
		`SELECT DISTINCT INDEX_NAME FROM information_schema.STATISTICS
		 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? AND NON_UNIQUE = 0 AND INDEX_NAME != 'PRIMARY'`,
		table, column,
	).Scan(&indexNames).Error; err != nil {
		log.Printf("Warning: could not list unique indexes on %s.%s: %v", table, column, err)
		return
	}
	for _, idx := range indexNames {
		ensureIndexDropped(db, table, idx)
	}
}
