package database

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// MigrateModels creates all tables in the database
func MigrateModels(db *gorm.DB) error {
	// Disable foreign key checks temporarily to avoid constraint conflicts during migration
	if err := db.Exec("SET FOREIGN_KEY_CHECKS=0").Error; err != nil {
		log.Printf("Warning: Could not disable foreign key checks: %v", err)
	}

	// egg.coop_id was created as BIGINT (GORM's default mapping for a bare Go `int`
	// before the explicit `type:int` tag existed on that field), while coop.coop_id is
	// INT. That mismatch made AutoMigrate's attempt to add the fk_coop_eggs constraint
	// below fail with error 3780 on every boot - and since AutoMigrate stops at the
	// first model that errors, everything after Egg in the list below never got
	// migrated either. Narrow it first so that FK can actually be created (see the
	// ensureForeignKey call after AutoMigrate).
	ensureColumnIsInt(db, "egg", "coop_id", false)

	// vaccine ("ประเภท" ยา/วัคซีน) และ vaccine_history (ประวัติให้จริงต่อคอก) เคยถูก
	// รวมเป็นตารางเดียวกันมาก่อน (แถว coop_id IS NULL = ประเภท, coop_id ไม่ว่าง =
	// ประวัติ) แต่ทำให้ผู้ใช้ดูตารางแล้วสับสนว่าแถวไหนเป็นอะไร จึงแยกกลับเป็น 2 ตาราง
	// ตามที่ผู้ใช้ขอ - ต้องรันก่อน AutoMigrate ทั้งก้อนด้านล่าง เพราะ models.Vaccine
	// ตอนนี้ประกาศ min_age_days/max_age_days เป็น NOT NULL แล้ว (ไม่ใช่ pointer แบบ
	// ตอนรวมตาราง) ถ้าแถวประวัติเก่า (ซึ่ง min_age_days เป็น NULL) ยังค้างอยู่ใน
	// vaccine ตอน AutoMigrate พยายามบังคับ NOT NULL จะ error ทันที
	migrateVaccineSplit(db)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Coop{},
		// Foodstock/ImportFood migrate right after Coop, before Device/SensorLog/Egg/Vaccine,
		// purely so their columns still get added first in case anything later in this list
		// ever fails again (AutoMigrate stops at the first model that errors) - egg/vaccine's
		// coop_id used to always fail here until the ensureColumnIsInt calls above started
		// narrowing them ahead of time; kept this ordering as cheap insurance either way.
		&models.Foodstock{},
		&models.ImportFood{},
		&models.FoodDistribution{},
		&models.FarmLayout{},
		&models.Device{},
		&models.SensorLog{},
		&models.Egg{},
		&models.Health{},
		&models.Vaccine{},
		&models.VaccineHistory{},
		&models.HealthAppointment{},
		&models.FarmThreshold{},
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
	// ⛔ เดิมตั้งเป็น unique เดี่ยวๆ แค่ name_coop ทำให้ชื่อคอกต้องไม่ซ้ำกับ "ทุก
	// farm ในระบบ" ทั้งที่ควรห้ามซ้ำแค่ภายใน farm (user) เดียวกันเท่านั้น - คนละ
	// user ตั้งชื่อคอกซ้ำกันได้ตามที่ผู้ใช้ขอ จึงเปลี่ยนเป็น unique คู่ (user_id,
	// name_coop) แทนด้านล่าง (หลัง ensureAdminBootstrapAndOwnership เติม user_id
	// ให้ครบก่อน) - ใช้ dropAllUniqueIndexesOnColumn (ไม่ใช่ ensureIndexDropped ตัว
	// เดียว) เพราะพบว่าคอลัมน์นี้มี unique index ซ้อนกันอยู่ 2 ตัวคนละชื่อในฐานจริง
	// (uq_coop_name_coop ที่ตั้งชื่อเอง กับอีกตัวชื่อ name_coop ที่ GORM สร้างเองแต่
	// ก่อน) ลบทีละชื่อจะตกหล่นตัวที่สอง
	dropAllUniqueIndexesOnColumn(db, "coop", "name_coop")

	// AutoMigrate above should now create these itself (coop_id is narrowed to INT before
	// it runs), but add them explicitly too as a guarded fallback - matching the naming
	// GORM itself used to attempt (from Coop's has-many Eggs/VaccineHistory associations),
	// so this is a no-op once AutoMigrate has already succeeded. ON DELETE CASCADE matches
	// device_ibfk_1/health_ibfk_1, the other two real FKs onto coop.
	ensureForeignKey(db, "egg", "fk_coop_eggs", "ALTER TABLE `egg` ADD CONSTRAINT `fk_coop_eggs` FOREIGN KEY (`coop_id`) REFERENCES `coop`(`coop_id`) ON DELETE CASCADE")
	ensureForeignKey(db, "vaccine_history", "fk_coop_vaccine_history", "ALTER TABLE `vaccine_history` ADD CONSTRAINT `fk_coop_vaccine_history` FOREIGN KEY (`coop_id`) REFERENCES `coop`(`coop_id`) ON DELETE CASCADE")

	// The name_coop FKs below turned out to be unreliable: app code never actually populates
	// name_coop on child rows (it's always left as ""), and multiple coops can have a blank
	// name too, so unrelated rows across different coops end up referencing the same empty
	// value. That breaks deleting any coop whose name_coop is "" - MySQL rejects it with a
	// FK violation even though DeleteCoop already cascades correctly via the real coop_id
	// relationships. Drop these FKs instead of re-creating them; coop_id is the source of truth.
	ensureForeignKeyDropped(db, "egg", "fk_name_coop_eggs")
	ensureForeignKeyDropped(db, "health", "fk_name_coop_healths")
	ensureForeignKeyDropped(db, "vaccine", "fk_name_coop_vaccines")
	ensureForeignKeyDropped(db, "device", "fk_name_coop_devices")

	// device.name used to be unique so it could double as a natural key for sensor_log.
	// That blocked placing more than one sensor of the same type in a coop. Drop every
	// unique index/FK still touching it, however it got created - the named uq_device_name
	// we added by hand, but also anything GORM's own AutoMigrate created automatically back
	// when the struct tag still said `unique` (that one isn't necessarily named the same).
	// (coop_id, slot_index) is the real identity for a placed device now.
	dropAllUniqueIndexesOnColumn(db, "device", "name")

	// foodstock used to have one global row per food_type (unique on food_type alone);
	// now it's one row per (food_type, user_id) instead, using a composite unique index
	// added via AutoMigrate's uniqueIndex tag above. Drop the old single-column unique
	// index left over from before - otherwise it collides the moment a second user gets
	// their own row for a food_type an existing row already used.
	ensureIndexDropped(db, "foodstock", "idx_foodstock_food_type")

	// เม็ดเล็ก/เม็ดใหญ่ ต้องมีแถวสต็อกของตัวเองเสมอ (ดึงยอดจากแถวเดิมแบบไม่มีประเภทมาไว้ที่เม็ดเล็ก)
	if err := EnsureFoodstockRows(); err != nil {
		log.Printf("Warning: could not ensure foodstock rows: %v", err)
	}

	// Bootstrap the first admin user (if none exists yet), backfill user_id on every
	// pre-existing root-entity row onto that admin, and lock the ownership column down
	// with a real FK - see ensureAdminBootstrapAndOwnership for the full breakdown.
	if err := ensureAdminBootstrapAndOwnership(db); err != nil {
		log.Printf("Warning: could not bootstrap admin/backfill ownership: %v", err)
	}

	// ชื่อคอกห้ามซ้ำแค่ภายใน farm (user) เดียวกัน - ทำหลัง ownership ด้านบน
	// เพื่อให้ user_id ของทุกแถวถูกเติมครบแล้วก่อนสร้าง index คู่นี้
	ensureUniqueIndex(db, "coop", "uq_coop_user_name", "ALTER TABLE `coop` ADD UNIQUE KEY `uq_coop_user_name` (`user_id`, `name_coop`)")

	fmt.Println("✓ All tables migrated successfully")
	return nil
}

// ownedTables lists every "root" entity table that got a user_id column added for the
// per-user data ownership feature (auth). Coop/Foodstock/ImportFood/FoodDistribution/
// FarmLayout/Vaccine each own their data directly; Egg/Health/VaccineHistory/Device are
// scoped indirectly through their parent coop instead (see database.CoopBelongsToUser)
// and so don't need a column or FK of their own here.
var ownedTables = []string{"coop", "foodstock", "importfood", "food_distribution", "farm_layout", "vaccine"}

// ensureAdminBootstrapAndOwnership is the one-time (but safe-to-rerun) migration step
// that turns on per-user data ownership on a database that predates it:
//  1. If the User table is empty, create a first account with a freshly generated random
//     password (logged once to stdout and written to
//     database/scripts/ADMIN_CREDENTIALS.local.txt - gitignored, local-machine only).
//  2. Backfill every pre-existing row in ownedTables that has no owner yet (user_id
//     IS NULL, i.e. it existed before this migration) onto the earliest account, so
//     nothing already in the live database becomes orphaned or inaccessible.
//  3. Add a real FK (user_id -> User.id_User, ON DELETE RESTRICT) on each of those
//     tables, using the same guarded ensureForeignKey helper as every other FK in this
//     file, so it's a no-op on every subsequent boot.
func ensureAdminBootstrapAndOwnership(db *gorm.DB) error {
	// GORM's AutoMigrate created user_id as BIGINT on every ownedTables (its default
	// mapping for a plain Go `int` field), even though the models now carry an explicit
	// `type:int` tag to match User.id_User's actual INT column - AutoMigrate doesn't
	// retroactively narrow an already-widened column type. Fix it up explicitly before
	// the FK below, which MySQL otherwise rejects with error 3780 (incompatible types).
	for _, table := range ownedTables {
		ensureColumnIsInt(db, table, "user_id", true)
	}

	adminID, err := ensureBootstrapAdmin(db)
	if err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}

	for _, table := range ownedTables {
		// Catches both NULL (the normal case - a pre-existing row that predates the
		// user_id column entirely) and 0 (a row inserted by older code between the
		// column being added and this backfill/FK running, before user_id was always
		// set on create) - either way it has no real owner yet.
		if err := db.Exec(fmt.Sprintf("UPDATE `%s` SET user_id = ? WHERE user_id IS NULL OR user_id = 0", table), adminID).Error; err != nil {
			log.Printf("Warning: could not backfill user_id on %s: %v", table, err)
		}
	}

	for _, table := range ownedTables {
		constraintName := "fk_" + table + "_user"
		ddl := fmt.Sprintf(
			"ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (`user_id`) REFERENCES `User`(`id_User`) ON DELETE RESTRICT",
			table, constraintName,
		)
		ensureForeignKey(db, table, constraintName, ddl)
	}

	return nil
}

// ensureBootstrapAdmin returns the id of the account that pre-existing, ownerless rows
// should be backfilled onto: the earliest registered user. There are no roles in this
// system, so "earliest account" is the only meaningful stand-in for "the farm's owner".
// If there are no users at all (a brand-new database), it creates one with a freshly
// generated random password. Safe to call on every boot.
func ensureBootstrapAdmin(db *gorm.DB) (int, error) {
	var existing models.User
	err := db.Order("id_User ASC").First(&existing).Error
	if err == nil {
		return existing.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, err
	}

	plainPassword, err := generateStrongPassword(20)
	if err != nil {
		return 0, fmt.Errorf("generate admin password: %w", err)
	}

	hashed, err := auth.HashPassword(plainPassword)
	if err != nil {
		return 0, fmt.Errorf("hash admin password: %w", err)
	}

	admin := models.User{
		Username: "admin",
		Password: hashed,
	}
	if err := db.Create(&admin).Error; err != nil {
		return 0, fmt.Errorf("create admin user: %w", err)
	}

	log.Println("=====================================================================")
	log.Println("✓ Bootstrapped the first admin user. THIS PASSWORD IS SHOWN ONLY ONCE:")
	log.Printf("    username: %s\n", admin.Username)
	log.Printf("    password: %s\n", plainPassword)
	log.Println("  (also saved to database/scripts/ADMIN_CREDENTIALS.local.txt)")
	log.Println("=====================================================================")

	if err := writeAdminCredentialsFile(admin.Username, plainPassword); err != nil {
		log.Printf("Warning: could not write ADMIN_CREDENTIALS.local.txt: %v", err)
	}

	return admin.ID, nil
}

// writeAdminCredentialsFile saves the freshly generated admin credentials to a local,
// gitignored file so they aren't lost if the startup log scrolls away. This is the only
// place the plaintext password is ever persisted.
func writeAdminCredentialsFile(username, password string) error {
	dir := filepath.Join("database", "scripts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(
		"EZ-SmartFarm bootstrap admin credentials\n"+
			"Generated automatically on first startup after the auth migration ran.\n"+
			"This file is gitignored (database/scripts/*.local.txt) - it never gets committed.\n\n"+
			"username: %s\n"+
			"password: %s\n",
		username, password,
	)

	return os.WriteFile(filepath.Join(dir, "ADMIN_CREDENTIALS.local.txt"), []byte(content), 0o600)
}

// generateStrongPassword returns a cryptographically random password of length n,
// drawn from a mixed letters+digits+symbols alphabet (crypto/rand, NOT math/rand).
func generateStrongPassword(n int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*-_"
	b := make([]byte, n)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = charset[idx.Int64()]
	}
	return string(b), nil
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

// ensureColumnIsInt narrows column on table to a plain INT if GORM's AutoMigrate left
// it as something else (typically BIGINT, its default mapping for a bare Go `int`
// field before an explicit `type:int` tag is added). A no-op once the column is
// already INT. nullable controls whether the narrowed column allows NULL - callers
// must pass the column's actual current nullability (MODIFY replaces the whole column
// definition, so getting this wrong would silently flip it).
func ensureColumnIsInt(db *gorm.DB, table, column string, nullable bool) {
	var dataType string
	if err := db.Raw(
		"SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		table, column,
	).Scan(&dataType).Error; err != nil {
		log.Printf("Warning: could not check column type %s.%s: %v", table, column, err)
		return
	}
	if dataType == "" || dataType == "int" {
		return
	}
	nullClause := "NOT NULL"
	if nullable {
		nullClause = "NULL"
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` INT %s", table, column, nullClause)).Error; err != nil {
		log.Printf("Warning: could not narrow column %s.%s to INT: %v", table, column, err)
	}
}

// ensureColumnDropped removes a column from table if it's still present. Used to
// clean up columns that no longer belong on a table after a schema restructure
// (e.g. vaccine's old coop_id/record_date columns once administration history
// moved to its own table - see migrateVaccineSplit). A no-op once already dropped.
func ensureColumnDropped(db *gorm.DB, table, column string) {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		table, column,
	).Scan(&count).Error; err != nil {
		log.Printf("Warning: could not check column %s.%s: %v", table, column, err)
		return
	}
	if count == 0 {
		return
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", table, column)).Error; err != nil {
		log.Printf("Warning: could not drop column %s.%s: %v", table, column, err)
	}
}

// migrationColumnExists reports whether table.column currently exists. Used by
// migrateVaccineSplit to tolerate running from a partially-completed prior attempt.
func migrationColumnExists(db *gorm.DB, table, column string) bool {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		table, column,
	).Scan(&count).Error; err != nil {
		log.Printf("Warning: could not check column %s.%s: %v", table, column, err)
		return false
	}
	return count > 0
}

// migrateVaccineSplit is the one-time cutover from the old single-table vaccine
// design (type rows with coop_id NULL + administration rows with coop_id set, all
// mixed in the "vaccine" table) to two separate tables: "vaccine" (types only) and
// "vaccine_history" (administration records only, see models/vaccine_history.go).
// Guarded on vaccine.coop_id still existing, so this only ever does real work once -
// every startup after that it's a single harmless COUNT query.
func migrateVaccineSplit(db *gorm.DB) {
	var coopIDColumnCount int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'vaccine' AND COLUMN_NAME = 'coop_id'",
	).Scan(&coopIDColumnCount).Error; err != nil {
		log.Printf("Warning: could not check vaccine.coop_id for split migration: %v", err)
		return
	}
	if coopIDColumnCount == 0 {
		return // already migrated
	}

	// 0. vaccine_history doesn't exist yet the very first time this runs (it's only
	// otherwise created by the main AutoMigrate call further down, which runs AFTER
	// this function) - create just this one table now so there's somewhere to copy
	// the administration rows into below.
	if err := db.AutoMigrate(&models.VaccineHistory{}); err != nil {
		log.Printf("Warning: could not create vaccine_history ahead of data migration: %v", err)
		return
	}

	// 1. copy every administration row (coop_id NOT NULL) into vaccine_history. This
	// whole function can fail partway through on any run (step 3 below drops several
	// columns one at a time and any single one of those Exec calls can fail without
	// aborting the rest), and the guard above only checks coop_id - so a prior run
	// may have already dropped record_date/name_coop/birthday/recommended_age while
	// still being considered "not done yet" on the next boot. Build the column list
	// from what's actually still there instead of assuming the full original
	// pre-split column set is present, or this INSERT fails forever on every restart
	// (as it did: "Unknown column 'record_date'" after an earlier run got that far).
	insertCols := []string{"coop_id", "name_vaccine", "method", "note"}
	selectExprs := []string{"coop_id", "name_vaccine", "method", "note"}
	optional := []struct {
		column, fallback string
	}{
		{"name_coop", "''"},
		{"birthday", "NULL"},
		{"record_date", "NOW()"},
		{"recommended_age", "''"},
	}
	for _, c := range optional {
		insertCols = append(insertCols, c.column)
		if migrationColumnExists(db, "vaccine", c.column) {
			selectExprs = append(selectExprs, c.column)
		} else {
			selectExprs = append(selectExprs, c.fallback)
		}
	}
	insertSQL := fmt.Sprintf(
		"INSERT INTO vaccine_history (%s) SELECT %s FROM vaccine WHERE coop_id IS NOT NULL",
		strings.Join(insertCols, ", "), strings.Join(selectExprs, ", "),
	)
	if err := db.Exec(insertSQL).Error; err != nil {
		log.Printf("Warning: could not copy vaccine administration rows into vaccine_history: %v", err)
		return
	}

	// 2. those rows now live in vaccine_history - remove them from vaccine so only
	// type rows remain there
	if err := db.Exec("DELETE FROM vaccine WHERE coop_id IS NOT NULL").Error; err != nil {
		log.Printf("Warning: could not remove migrated rows from vaccine: %v", err)
		return
	}

	// 3. drop the FK and the columns that only ever applied to administration rows -
	// vaccine is now types-only (name_vaccine/method/note/min_age_days/max_age_days)
	ensureForeignKeyDropped(db, "vaccine", "fk_coop_vaccines")
	ensureColumnDropped(db, "vaccine", "coop_id")
	ensureColumnDropped(db, "vaccine", "name_coop")
	ensureColumnDropped(db, "vaccine", "birthday")
	ensureColumnDropped(db, "vaccine", "record_date")
	ensureColumnDropped(db, "vaccine", "recommended_age")
	// legacy duplicate of name_vaccine left over from a much older schema - dead
	// weight even before this split, dropping it now since we're already here
	ensureColumnDropped(db, "vaccine", "name")

	// 4. every remaining row is a type row now, so min_age_days/max_age_days are
	// always populated - narrow them to a plain NOT NULL INT to match models.Vaccine
	// (ensureColumnIsInt is a no-op once already INT, so this is safe to leave in
	// permanently rather than only inside this guarded block)
	ensureColumnIsInt(db, "vaccine", "min_age_days", false)
	ensureColumnIsInt(db, "vaccine", "max_age_days", false)

	log.Printf("✓ Split vaccine into vaccine (types) + vaccine_history (administration records)")
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
