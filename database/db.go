// =====================================================================================
// ไฟล์ database/db.go - เปิด/ปิดการเชื่อมต่อฐานข้อมูล MySQL
// =====================================================================================
//
// โฟลเดอร์ database/ คือ "ชั้นคุยกับฐานข้อมูล" (repository layer) handler จะไม่เขียน SQL เอง
// แต่เรียกฟังก์ชันในโฟลเดอร์นี้แทน เช่น database.GetAllCoops(userID)
// แยกแบบนี้เพื่อให้ handler ดูแลแค่เรื่องรับ/ตอบ HTTP ส่วนเรื่อง query อยู่ที่เดียว แก้ง่าย
//
// ใช้ GORM = ORM (Object Relational Mapping) ของ Go แปลง struct ใน models/ <-> แถวในตาราง
// เขียน DB.Where(...).Find(&rows) แทน SQL ดิบ ลดโอกาสโดน SQL injection เพราะ GORM
// ใส่ค่าผ่าน placeholder (?) ให้อัตโนมัติ
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"EZ-SmartFarm_BachN/config"
	"EZ-SmartFarm_BachN/models" // 🌟 1. เพิ่ม import models เข้ามาเพื่อให้ไฟล์นี้รู้จักโครงสร้างตาราง

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB = ตัวเชื่อมต่อฐานข้อมูลตัวเดียวที่ใช้ร่วมกันทั้งโปรแกรม (ตั้งค่าใน InitDatabase)
// GORM มี connection pool ในตัว จึงแชร์ตัวเดียวได้ปลอดภัยแม้มีหลาย request พร้อมกัน
// ไฟล์อื่นในแพ็กเกจ database ใช้ DB ตรง ๆ ส่วนแพ็กเกจอื่นเรียกผ่าน GetDB()
var DB *gorm.DB

// InitDatabase เปิดการเชื่อมต่อ MySQL ตามค่าใน config แล้วเก็บไว้ในตัวแปร DB
//
// ขั้นตอน:
//  1. cfg.GetDSN() ประกอบข้อความเชื่อมต่อ (config/config.go)
//  2. ตั้ง logger ของ GORM: แสดงเฉพาะระดับ Warn ขึ้นไป, เตือนเมื่อ query ช้ากว่า 200ms
//     และไม่ต้อง log "record not found" (บางที่ใช้กรณีนี้เป็นสถานะปกติ เช่นผู้ใช้ยังไม่ตั้งค่า)
//  3. gorm.Open(...) ต่อฐานข้อมูล ถ้าไม่สำเร็จคืน error ให้ main.go เขียนเตือน
//  4. AutoMigrate ตาราง Coop, Vaccine, VaccineHistory ให้ตรงกับ struct ใน models/
//     (ตารางอื่นสร้างใน migration.go -> MigrateModels ที่ main.go เรียกต่อ)
//
// ถูกเรียกจาก: main.go
func InitDatabase(cfg config.DatabaseConfig) error {
	dsn := cfg.GetDSN()
	log.Printf("DEBUG: connecting to host=%s port=%s db=%s user=%s", cfg.Host, cfg.Port, cfg.Database, cfg.User)

	// IgnoreRecordNotFoundError: true - หลายที่ใน repository layer (เช่น
	// GetFarmThreshold, farm_layout) ใช้ "record not found" เป็นสถานะปกติ
	// (ผู้ใช้ยังไม่เคยตั้งค่า ไม่ใช่ error จริง) ค่า default ของ GORM log ระดับ
	// Error/สีแดงทุกครั้งที่เจอ ทำให้ log บน Render ดูเหมือนมีบั๊กทั้งที่ไม่มี
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}

	DB = db
	fmt.Println("Database connected successfully")

	// 🌟 2. เพิ่มคำสั่ง AutoMigrate ตรงนี้ เพื่อให้ GORM สร้างตารางให้อัตโนมัติ
	// นำชื่อตารางที่คุณต้องการให้มันสร้าง/อัปเดตอัตโนมัติมาใส่ไว้ในวงเล็บ
	err = DB.AutoMigrate(
		&models.Coop{},           // ตารางคอกไก่
		&models.Vaccine{},        // ตาราง "ประเภท" ยา/วัคซีน (เกณฑ์ช่วงอายุ ไม่ผูกคอก)
		&models.VaccineHistory{}, // ประวัติการให้วัคซีนจริงของแต่ละคอก (แยกจากตารางข้างบน)
		// ถัามีโมเดลอื่นๆ ในโฟลเดอร์ models เช่น Egg, Device ก็สามารถเพิ่มต่อท้ายได้เลยครับ
	)

	if err != nil {
		log.Printf("Failed to auto migrate database tables: %v", err)
	} else {
		fmt.Println("Database tables migrated successfully")
	}

	return nil
}

// GetDB คืนตัวเชื่อมต่อ DB ให้แพ็กเกจอื่นใช้ (handlers/, services/, routes/, main.go)
func GetDB() *gorm.DB {
	return DB
}

// CloseDatabase ปิดการเชื่อมต่อฐานข้อมูลให้เรียบร้อย ถูกเรียกจาก main.go ตอนได้รับสัญญาณปิดเซิร์ฟเวอร์
// ถ้ายังไม่เคยต่อ (DB เป็น nil) ก็ไม่ต้องทำอะไร
// DB.DB() ดึงตัวเชื่อมต่อระดับล่าง (database/sql) ออกมาเพื่อสั่ง Close
func CloseDatabase() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
