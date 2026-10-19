// =====================================================================================
// ไฟล์ config/config.go - อ่านค่าตั้งค่าของระบบ (ที่อยู่ฐานข้อมูล, พอร์ตเซิร์ฟเวอร์)
// =====================================================================================
//
// ทำไมไม่เขียนรหัสผ่าน DB ลงในโค้ดตรง ๆ: เพราะโค้ดขึ้น GitHub ใครก็อ่านได้ และเครื่องพัฒนา
// กับเซิร์ฟเวอร์จริง (Render) ใช้ DB คนละตัว จึงเก็บค่าไว้ใน environment variable แทน
//  - ตอนรันในเครื่อง: อ่านจากไฟล์ .env (ไม่ขึ้น git)
//  - ตอนรันบน Render: ตั้งค่าในหน้า Environment ของ Render
// ถูกเรียกจาก: main.go -> config.LoadConfig()
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// DatabaseConfig เก็บข้อมูลที่ใช้ต่อ MySQL: host, port, ชื่อผู้ใช้, รหัสผ่าน, ชื่อฐานข้อมูล
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// Config คือค่าตั้งค่าทั้งหมดของโปรแกรม: ค่าฐานข้อมูล + พอร์ตที่เซิร์ฟเวอร์เปิดรอ
type Config struct {
	Database   DatabaseConfig
	ServerPort string
}

// LoadConfig อ่านค่าตั้งค่าทั้งหมดแล้วคืนเป็น Config
//
// ขั้นตอน:
//  1. godotenv.Load() โหลดไฟล์ .env เข้าสู่ environment (ถ้าไม่มีไฟล์ เช่นบน Render ก็ข้ามไป ไม่ error)
//  2. อ่านแต่ละค่าผ่าน getEnv ถ้าไม่ได้ตั้งไว้จะใช้ค่าเริ่มต้น เช่น DB_HOST = localhost, SERVER_PORT = 8080
// ถูกเรียกจาก: main.go บรรทัดแรกของ main()
func LoadConfig() Config {
	// Load .env file (ignore error if not exists)
	_ = godotenv.Load()

	return Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "ez_smartfarm"),
		},
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

// GetDSN ประกอบ "DSN" (Data Source Name) = ข้อความที่บอกไดรเวอร์ MySQL ว่าจะต่อที่ไหน
// รูปแบบ: user:password@tcp(host:port)/ชื่อDB?ตัวเลือก
//
// ตัวเลือกที่ใส่:
//  - charset=utf8mb4 รองรับภาษาไทยและอีโมจิครบ (utf8 ธรรมดาของ MySQL เก็บอีโมจิไม่ได้)
//  - parseTime=True  ให้คอลัมน์ DATE/DATETIME แปลงเป็น time.Time ของ Go อัตโนมัติ
//  - loc=Local       ใช้เขตเวลาของเครื่องเซิร์ฟเวอร์
// ถูกเรียกจาก: database/db.go -> InitDatabase
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// getEnv อ่าน environment variable ชื่อ key ถ้าว่างหรือไม่มี คืน defaultValue แทน
// ใช้ภายในไฟล์นี้เท่านั้น (ชื่อขึ้นต้นตัวเล็ก = private ในภาษา Go)
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
