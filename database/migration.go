// =====================================================================================
// ไฟล์ database/migration.go - สร้าง/ปรับโครงสร้างตารางอัตโนมัติทุกครั้งที่เปิดเซิร์ฟเวอร์
// =====================================================================================
//
// Migration = การทำให้โครงสร้างฐานข้อมูลตรงกับ model ในโค้ด (models/*.go)
// ทุกครั้งที่เซิร์ฟเวอร์เริ่ม main.go จะเรียก MigrateModels
//
// หลักการสำคัญของไฟล์นี้: ทุกขั้นตอน "รันซ้ำได้ปลอดภัย" (idempotent)
// คือเช็กก่อนว่าทำไปแล้วหรือยัง ถ้าทำแล้วก็ข้าม จึงรันทุกครั้งที่เปิดเครื่องได้โดยไม่พัง
// ฟังก์ชันชื่อ ensure... = "ทำให้แน่ใจว่ามี/เป็นแบบนี้" ส่วน ...Dropped = "ทำให้แน่ใจว่าลบไปแล้ว"
//
// ส่วนใหญ่เป็นการแก้ปัญหาฐานข้อมูลรุ่นเก่าที่สร้างไว้ก่อนแก้โค้ด (คอลัมน์ผิดชนิด, index เก่า, ตารางที่ถูกแยก)
// ทำให้ฐานข้อมูลเดิมที่ใช้งานจริงอัปเกรดได้โดยไม่ต้องลบข้อมูลทิ้ง
package database

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/models"

	"gorm.io/gorm"
)

// MigrateModels จุดเริ่ม migration ทั้งหมด เรียกครั้งเดียวตอนเปิดเซิร์ฟเวอร์จาก main.go
//
// ลำดับการทำงาน:
//  1. ปิดการเช็ก Foreign Key ชั่วคราว (กันการปรับตารางชนกัน)
//  2. แก้คอลัมน์รุ่นเก่าก่อน AutoMigrate (egg.coop_id เป็น INT, แยกตารางวัคซีน)
//  3. AutoMigrate ทุก model = GORM สร้างตาราง/เพิ่มคอลัมน์ที่ขาดให้อัตโนมัติ
//     ถ้าล้มกลางทาง แค่ log เตือนแล้วทำต่อ เพราะขั้นต่อไปยังจำเป็นต้องรัน
//  4. เปิดการเช็ก Foreign Key กลับ
//  5. ขยายคอลัมน์ไอคอนเป็น TEXT (ไอคอน SVG ยาวเกิน 255 ตัวอักษร)
//     และ kg_given เป็นทศนิยม 4 ตำแหน่ง (ไก่เล็กกินวันละ ~0.0006 กก. ถ้า 2 ตำแหน่งจะเป็น 0.00)
//  6. จัดการ index / Foreign Key: เพิ่มที่ต้องมี ลบที่ไม่ใช้แล้ว
//  7. สร้างแถวสต็อกอาหาร และกำหนดเจ้าของให้ข้อมูลเก่า (ensureAdminBootstrapAndOwnership)
//  8. เพิ่ม unique key: ชื่อคอกห้ามซ้ำในผู้ใช้เดียวกัน, ผลตรวจสุขภาพ 1 แถว/คอก/วัน,
//     เลขอุปกรณ์ห้ามซ้ำในคอกเดียวกัน
//
// ทำไมลำดับสำคัญ: บางขั้นต้องทำหลังขั้นอื่น เช่น unique (user_id, name_coop) ต้องรอเติม user_id ให้ครบก่อน
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
		&models.FoodAgeRule{},
		&models.DeviceType{},
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

	// device_type.icon was originally created as VARCHAR(255) by AutoMigrate (its
	// mapping before the struct tag changed to `text`) - SVG data: URI icons run
	// 500+ characters, so every INSERT into this column was silently rejected by
	// MySQL strict mode.
	ensureColumnIsText(db, "device_type", "icon")
	// device.icon มีปัญหาเดียวกัน - คัดลอกค่าไอคอน (SVG data URI) มาจาก device_type
	// ตอนวางอุปกรณ์ลงคอก ถ้าคอลัมน์ยังแคบอยู่จะ insert/update ไม่ผ่านแบบเงียบๆ เหมือนกัน
	ensureColumnIsText(db, "device", "icon")

	// food_distribution.kg_given was DECIMAL(10,2): one chick eats ~0.0006 kg/day, so small
	// coops were stored as 0.00. Widen to 4 decimals (existing values are kept as-is).
	ensureDecimalScale(db, "food_distribution", "kg_given", "DECIMAL(10,4) NOT NULL", 4)

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

	// ผลตรวจสุขภาพห้ามมีมากกว่า 1 แถวต่อคอกต่อวัน (เดิมไม่มีการกันเลย ยิง POST
	// /api/healths ซ้ำกี่ครั้งก็สร้างแถวใหม่ได้เรื่อยๆ - ต้องเคลียร์แถวซ้ำที่มีอยู่
	// แล้วในฐานก่อน ไม่งั้น ALTER TABLE ด้านล่างจะชนแล้วไม่ถูกสร้างแบบเงียบๆ)
	dedupeHealthRecords(db)
	ensureUniqueIndex(db, "health", "uq_health_coop_date", "ALTER TABLE `health` ADD UNIQUE KEY `uq_health_coop_date` (`coop_id`, `date`)")

	// ตัวนับเลขอุปกรณ์ต่อคอก (coop.next_device_no): ดันให้เกินเลขสูงสุดที่มีอยู่เสมอ (ทำซ้ำได้ ไม่ย้อนลง) เพราะคอก
	// ที่มีอุปกรณ์อยู่ก่อนมีคอลัมน์นี้ต้องไม่ได้เลขซ้ำกับของเดิม แล้วกันเลขอุปกรณ์ซ้ำในคอกเดียวกันที่ระดับฐานข้อมูล
	if err := db.Exec("UPDATE coop c SET c.next_device_no = GREATEST(c.next_device_no, COALESCE((SELECT MAX(d.slot_index) + 1 FROM device d WHERE d.coop_id = c.coop_id), 1))").Error; err != nil {
		log.Printf("Warning: could not backfill coop.next_device_no: %v", err)
	}
	ensureUniqueIndex(db, "device", "uq_device_coop_slot", "ALTER TABLE `device` ADD UNIQUE KEY `uq_device_coop_slot` (`coop_id`, `slot_index`)")

	// เวลาเก่าที่เขียนตอน backend ยังใช้เวลา UTC ให้เลื่อนเป็นเวลาไทย (+7 ชม.) ครั้งเดียว
	shiftUTCDatetimesToBangkok(db)

	fmt.Println("✓ All tables migrated successfully")
	return nil
}

// datetimeColumnsWrittenInUTC คอลัมน์ชนิด DATETIME ที่ backend เขียนลงไป ตอนยังรันด้วยเวลา UTC บน Render
// (คอลัมน์ชนิด DATE เก็บแค่วันที่ ไม่มีเวลา จึงไม่อยู่ในรายการนี้)
var datetimeColumnsWrittenInUTC = [][2]string{
	{"User", "created_at"}, {"User", "updated_at"},
	{"coop", "date_adopt_animals"}, {"coop", "birthday"},
	{"device", "last_update"},
	{"egg", "date_collect_egg"},
	{"food_distribution", "distributed_at"},
	{"health", "date"},
	{"health_appointment", "created_at"},
	{"importfood", "import_date"},
	{"sensor_log", "timestamp"},
}

// shiftUTCDatetimesToBangkok เลื่อนเวลาเก่าทุกคอลัมน์ใน datetimeColumnsWrittenInUTC ไป +7 ชั่วโมง "ครั้งเดียว"
//
// ทำไมต้องมี: เดิม backend บน Render ใช้เวลา UTC เวลาที่ลงฐานข้อมูลจึงช้ากว่าเวลาไทย 7 ชั่วโมง
// (เช่น สร้างบัญชีตอน 13:36 แต่ created_at เป็น 06:36) ตอนนี้ main.go ตั้ง time.Local เป็นเวลาไทยแล้ว
// แถวใหม่จึงถูกต้อง ฟังก์ชันนี้แก้แถวเก่าให้เป็นเวลาไทยเหมือนกัน
//
// กันทำซ้ำ: บันทึกชื่อ migration ลงตาราง app_migration ถ้าเคยทำแล้วจะข้ามทันที
// ทำใน Transaction เดียว ถ้าคอลัมน์ไหนล้ม ย้อนกลับทั้งหมด และไม่บันทึกว่าทำแล้ว (ครั้งหน้าจะลองใหม่)
// ข้ามค่าว่าง / ค่าศูนย์ (ก่อนปี 1900) เพราะเป็นค่า "ไม่มีวันที่" ถ้าบวกเวลาเข้าไปจะกลายเป็นวันที่จริง
// ตรวจชนิดคอลัมน์จริงก่อน ถ้าไม่ใช่ DATETIME (หรือไม่มีคอลัมน์นั้น) จะข้าม
//
// ข้อควรรู้: แถวที่เคยเขียนจากเครื่องที่ตั้งเวลาไทยอยู่แล้ว (รัน go run . ในเครื่อง) จะถูกบวกไปด้วย
// คอลัมน์ที่เป็นวันที่ (วันเกิดไก่ วันเก็บไข่ ฯลฯ) ไม่กระทบวันที่ที่แสดง ส่วนคอลัมน์ที่เป็นเวลา จะเลื่อนไป 7 ชม.
// ถูกเรียกจาก: MigrateModels
func shiftUTCDatetimesToBangkok(db *gorm.DB) {
	const name = "2026-10-19_shift_utc_datetimes_to_bangkok"
	if err := db.Exec("CREATE TABLE IF NOT EXISTS `app_migration` (`name` VARCHAR(100) NOT NULL PRIMARY KEY, `applied_at` DATETIME NOT NULL)").Error; err != nil {
		log.Printf("Warning: could not create app_migration table: %v", err)
		return
	}
	var done int64
	if err := db.Raw("SELECT COUNT(*) FROM `app_migration` WHERE `name` = ?", name).Scan(&done).Error; err != nil || done > 0 {
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, tc := range datetimeColumnsWrittenInUTC {
			table, column := tc[0], tc[1]
			var dataType string
			if err := tx.Raw("SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?", table, column).Scan(&dataType).Error; err != nil {
				return err
			}
			if dataType != "datetime" {
				continue
			}
			res := tx.Exec(fmt.Sprintf("UPDATE `%s` SET `%s` = `%s` + INTERVAL 7 HOUR WHERE `%s` > '1900-01-01'", table, column, column, column))
			if res.Error != nil {
				return fmt.Errorf("%s.%s: %w", table, column, res.Error)
			}
			log.Printf("  shifted %s.%s +7h (%d rows)", table, column, res.RowsAffected)
		}
		return tx.Exec("INSERT INTO `app_migration` (`name`, `applied_at`) VALUES (?, ?)", name, time.Now()).Error
	})
	if err != nil {
		log.Printf("Warning: could not shift UTC datetimes to Bangkok time: %v", err)
		return
	}
	fmt.Println("✓ Shifted old UTC datetimes to Thai time (+7h)")
}

// ownedTables รายชื่อตาราง "ระดับบนสุด" ที่มีคอลัมน์ user_id ระบุเจ้าของโดยตรง
// ตารางลูก (egg, health, vaccine_history, device) ไม่อยู่ในนี้ เพราะเช็กเจ้าของผ่านคอกแม่
// (ดู CoopBelongsToUser ใน ownership.go)
// ถูกใช้ใน: ensureAdminBootstrapAndOwnership
var ownedTables = []string{"coop", "foodstock", "importfood", "food_distribution", "farm_layout", "vaccine", "device_type"}

// ensureAdminBootstrapAndOwnership เปิดระบบ "ข้อมูลแยกตามผู้ใช้" ให้ฐานข้อมูลรุ่นเก่าที่ยังไม่มีเจ้าของ
//  1. ถ้ายังไม่มีผู้ใช้เลย สร้างบัญชีแรกพร้อมรหัสผ่านสุ่ม (แสดงใน log ครั้งเดียว และเขียนลงไฟล์ในเครื่อง)
//  2. ข้อมูลเก่าในตาราง ownedTables ที่ user_id ยังว่าง ให้เป็นของบัญชีแรก
//     กันข้อมูลเดิมกลายเป็น "ไม่มีเจ้าของ" จนไม่มีใครเห็น
//  3. เพิ่ม Foreign Key user_id -> User (ON DELETE RESTRICT = ห้ามลบผู้ใช้ที่ยังมีข้อมูลอยู่)
//
// รันซ้ำได้: ครั้งต่อไปไม่มีแถวที่ user_id ว่าง และ FK มีแล้ว จึงไม่ทำอะไร
// ถูกเรียกจาก: MigrateModels
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

// ensureBootstrapAdmin หา id ของบัญชีที่จะเป็นเจ้าของข้อมูลเก่า = ผู้ใช้คนแรกที่สมัคร (id น้อยสุด)
// ระบบไม่มีการแบ่ง role จึงถือว่าบัญชีแรกคือเจ้าของฟาร์ม
// ถ้าฐานใหม่ยังไม่มีผู้ใช้ จะสร้างบัญชีให้ด้วยรหัสผ่านสุ่ม (generateStrongPassword) และ hash ด้วย bcrypt
// ถูกเรียกจาก: ensureAdminBootstrapAndOwnership
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

// writeAdminCredentialsFile บันทึกชื่อผู้ใช้/รหัสผ่านของบัญชีแรกลงไฟล์
// database/scripts/ADMIN_CREDENTIALS.local.txt (ไฟล์นี้อยู่ใน .gitignore ไม่ถูกอัปขึ้น git)
// ทำไม: กันรหัสผ่านหายถ้า log เลื่อนผ่านไป เป็นที่เดียวที่เก็บรหัสผ่านแบบอ่านออก
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

// generateStrongPassword สร้างรหัสผ่านสุ่มยาว n ตัว จากตัวอักษร+ตัวเลข+สัญลักษณ์
// ใช้ crypto/rand (สุ่มแบบปลอดภัยสำหรับงานความปลอดภัย) ไม่ใช้ math/rand ที่เดาลำดับได้
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

// ensureUniqueIndex เพิ่ม unique index ถ้ายังไม่มี (เช็กจาก information_schema ก่อน)
// ถ้าเพิ่มไม่สำเร็จ (เช่นมีข้อมูลซ้ำอยู่แล้ว) แค่ log เตือน ไม่หยุดเซิร์ฟเวอร์
// ทำไมต้องเช็กก่อน: migration รันทุกครั้งที่เปิดเครื่อง ถ้าสั่ง ADD ซ้ำจะ error
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

// dedupeHealthRecords ลบผลตรวจสุขภาพที่ซ้ำกัน (คอกเดียวกัน วันเดียวกัน) ให้เหลือแถวล่าสุด (health_id สูงสุด)
// ต้องทำก่อนเพิ่ม unique key uq_health_coop_date
// ทำไม: ถ้ามีข้อมูลซ้ำค้างอยู่ การเพิ่ม unique key จะล้ม และ ensureUniqueIndex แค่ log เตือน
// ผลคือจะไม่มีอะไรกันบันทึกซ้ำได้จริง
// ถูกเรียกจาก: MigrateModels
func dedupeHealthRecords(db *gorm.DB) {
	type dupGroup struct {
		CoopID int
		Date   time.Time
	}
	var groups []dupGroup
	if err := db.Raw(
		"SELECT coop_id, date FROM health GROUP BY coop_id, date HAVING COUNT(*) > 1",
	).Scan(&groups).Error; err != nil {
		log.Printf("Warning: could not check for duplicate health records: %v", err)
		return
	}
	for _, g := range groups {
		var keepID int
		if err := db.Raw(
			"SELECT health_id FROM health WHERE coop_id = ? AND date = ? ORDER BY health_id DESC LIMIT 1",
			g.CoopID, g.Date,
		).Scan(&keepID).Error; err != nil {
			log.Printf("Warning: could not pick row to keep for coop %d date %v: %v", g.CoopID, g.Date, err)
			continue
		}
		res := db.Exec(
			"DELETE FROM health WHERE coop_id = ? AND date = ? AND health_id != ?",
			g.CoopID, g.Date, keepID,
		)
		if res.Error != nil {
			log.Printf("Warning: could not delete duplicate health records for coop %d date %v: %v", g.CoopID, g.Date, res.Error)
			continue
		}
		log.Printf("Dedup: removed %d duplicate health record(s) for coop_id=%d date=%v (kept health_id=%d)", res.RowsAffected, g.CoopID, g.Date, keepID)
	}
}

// ensureForeignKey เพิ่ม Foreign Key ถ้ายังไม่มี (เช็กจาก information_schema ก่อน)
// Foreign Key = บังคับว่าค่าในคอลัมน์ต้องมีอยู่จริงในตารางที่อ้างถึง เช่น egg.coop_id ต้องเป็นคอกที่มีจริง
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

// ensureColumnIsInt เปลี่ยนชนิดคอลัมน์เป็น INT ถ้ายังไม่ใช่ (ปกติเป็น BIGINT ที่ GORM สร้างไว้รุ่นก่อน)
// ทำไม: Foreign Key ต้องชนิดตรงกันทั้งสองฝั่ง coop.coop_id เป็น INT ถ้าอีกฝั่งเป็น BIGINT จะสร้าง FK ไม่ได้
// nullable ต้องส่งให้ตรงกับคอลัมน์จริง เพราะคำสั่ง MODIFY เขียนทับนิยามคอลัมน์ทั้งหมด
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

// ensureColumnIsText ขยายคอลัมน์จาก VARCHAR(255) เป็น TEXT ถ้ายังไม่ใช่
// ทำไม: AutoMigrate เพิ่มคอลัมน์ใหม่ได้ แต่ไม่ขยายคอลัมน์เดิมให้ ไอคอน SVG ยาว 500+ ตัวอักษร
// ถ้ายังเป็น 255 การบันทึกจะล้ม
func ensureColumnIsText(db *gorm.DB, table, column string) {
	var dataType string
	if err := db.Raw(
		"SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		table, column,
	).Scan(&dataType).Error; err != nil {
		log.Printf("Warning: could not check column type %s.%s: %v", table, column, err)
		return
	}
	if dataType == "" || strings.Contains(dataType, "text") {
		return
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` TEXT NOT NULL", table, column)).Error; err != nil {
		log.Printf("Warning: could not widen column %s.%s to TEXT: %v", table, column, err)
	}
}

// ensureDecimalScale ขยายคอลัมน์ DECIMAL ให้มีทศนิยมอย่างน้อย wantScale ตำแหน่ง ถ้ามีครบแล้วก็ข้าม
// ใช้กับ food_distribution.kg_given (2 -> 4 ตำแหน่ง) ค่าเดิมยังอยู่ครบ
func ensureDecimalScale(db *gorm.DB, table, column, definition string, wantScale int) {
	var scale *int
	if err := db.Raw(
		"SELECT NUMERIC_SCALE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		table, column,
	).Scan(&scale).Error; err != nil {
		log.Printf("Warning: could not check column scale %s.%s: %v", table, column, err)
		return
	}
	if scale == nil || *scale >= wantScale {
		return
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN `%s` %s", table, column, definition)).Error; err != nil {
		log.Printf("Warning: could not widen column %s.%s: %v", table, column, err)
	}
}

// ensureColumnDropped ลบคอลัมน์ที่ไม่ใช้แล้ว ถ้ายังมีอยู่
// เช่น vaccine.coop_id / record_date หลังย้ายประวัติวัคซีนไปตาราง vaccine_history แล้ว
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

// migrationColumnExists ตอบว่าตารางนี้มีคอลัมน์นี้อยู่หรือไม่ (ดูจาก information_schema)
// ใช้ใน migrateVaccineSplit เพื่อรันต่อได้แม้ครั้งก่อนทำค้างไว้ครึ่งทาง
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

// migrateVaccineSplit แยกตารางวัคซีนรุ่นเก่า (ตารางเดียว) เป็น 2 ตาราง
//   - รุ่นเก่า: ตาราง vaccine เก็บทั้ง "ประเภท" (coop_id ว่าง) และ "ประวัติการให้" (coop_id มีค่า) ปนกัน
//   - รุ่นใหม่: vaccine = ประเภทอย่างเดียว, vaccine_history = ประวัติการให้อย่างเดียว
//
// ขั้นตอน: คัดลอกแถวประวัติไป vaccine_history -> ลบแถวประวัติออกจาก vaccine -> ลบคอลัมน์ที่ไม่ใช้
// ทำจริงแค่ครั้งเดียว: เช็กว่า vaccine.coop_id ยังอยู่ไหม ถ้าไม่อยู่แล้ว = แยกเสร็จแล้ว ข้าม
// ทำไมแยก: ผู้ใช้เปิดดูตารางแล้วสับสนว่าแถวไหนเป็นประเภท แถวไหนเป็นประวัติ
// ถูกเรียกจาก: MigrateModels (ก่อน AutoMigrate)
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

// ensureIndexDropped ลบ index ที่ไม่ใช้แล้วถ้ายังมีอยู่ (ใช้ถอน constraint ที่ migration รุ่นก่อนเคยเพิ่ม)
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

// ensureForeignKeyDropped ลบ Foreign Key ที่ไม่ใช้แล้วถ้ายังมีอยู่
// เช่น FK ที่อ้างด้วย name_coop ซึ่งทำให้ลบคอกที่ชื่อว่างไม่ได้ (ใช้ coop_id เป็นตัวอ้างอิงจริงแทน)
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

// dropAllUniqueIndexesOnColumn ลบ unique index "ทุกตัว" ที่ครอบคอลัมน์นี้ (ยกเว้น PRIMARY)
// และลบ Foreign Key ที่อ้างคอลัมน์นี้ก่อน เพราะ MySQL ไม่ยอมลบ index ที่ FK ยังใช้อยู่
// ทำไมไม่ลบตามชื่อ: GORM เคยสร้าง unique index ด้วยชื่อที่เราไม่ได้ตั้งเอง
// ในฐานจริงพบ 2 ตัวคนละชื่อบนคอลัมน์เดียวกัน ลบทีละชื่อจะตกหล่น
// ใช้กับ: coop.name_coop และ device.name
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
