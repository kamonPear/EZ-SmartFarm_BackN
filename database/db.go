package database

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/config"
	"EZ-SmartFarm_BachN/models" // 🌟 1. เพิ่ม import models เข้ามาเพื่อให้ไฟล์นี้รู้จักโครงสร้างตาราง

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDatabase(cfg config.DatabaseConfig) error {
	dsn := cfg.GetDSN()
	log.Printf("DEBUG: connecting to host=%s port=%s db=%s user=%s", cfg.Host, cfg.Port, cfg.Database, cfg.User)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return err
	}

	DB = db
	fmt.Println("Database connected successfully")

	// 🌟 2. เพิ่มคำสั่ง AutoMigrate ตรงนี้ เพื่อให้ GORM สร้างตารางให้อัตโนมัติ
	// นำชื่อตารางที่คุณต้องการให้มันสร้าง/อัปเดตอัตโนมัติมาใส่ไว้ในวงเล็บ
	err = DB.AutoMigrate(
		&models.MedicineSchedule{}, // 👈 ตารางยาและวัคซีนใหม่ของเรา
		&models.Coop{},             // ตารางคอกไก่
		&models.Vaccine{},          // ตารางประวัติการฉีดวัคซีน
		// ถัามีโมเดลอื่นๆ ในโฟลเดอร์ models เช่น Egg, Device ก็สามารถเพิ่มต่อท้ายได้เลยครับ
	)
	
	if err != nil {
		log.Printf("Failed to auto migrate database tables: %v", err)
	} else {
		fmt.Println("Database tables migrated successfully")
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}

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