package scheduler

import (
	"log"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
	"github.com/robfig/cron/v3"
)

// SetupJobs สำหรับตั้งเวลาการทำงานอัตโนมัติ
func SetupJobs() {
	// สร้าง cron instance
	c := cron.New()

	// ตั้งเวลา: ทำงานทุกวัน เวลา 00:00 น. (เที่ยงคืน)
	// เปลี่ยนเป็น "* * * * *" ถ้าต้องการทดสอบให้ทำงานทุกๆ 1 นาที
	_, err := c.AddFunc("0 0 * * *", func() {
		log.Println("⏰ [Cron] กำลังตัดสต็อกอาหารประจำวัน (เม็ดเล็ก 20 kg, เม็ดใหญ่ 30 kg)...")

		// ตัดสต็อกแต่ละประเภทแยกกัน คนละอัตราต่อวัน ให้ทุกผู้ใช้ในระบบ (cron ไม่มี
		// ผู้ใช้ที่ล็อกอินอยู่ให้ผูก context ด้วย จึงตัดสต็อกของทุกบัญชีเท่ากัน)
		if err := database.DeductFoodstockByTypeAllUsers(models.FoodTypeSmallPellet, 20.0); err != nil {
			log.Printf("❌ [Cron] ตัดสต็อกเม็ดเล็กล้มเหลว: %v\n", err)
		}
		if err := database.DeductFoodstockByTypeAllUsers(models.FoodTypeLargePellet, 30.0); err != nil {
			log.Printf("❌ [Cron] ตัดสต็อกเม็ดใหญ่ล้มเหลว: %v\n", err)
		}
	})

	if err != nil {
		log.Fatalf("ตั้งค่า Cron Job ล้มเหลว: %v", err)
	}

	// เริ่มการทำงาน
	c.Start()
	log.Println("✅ [Cron] Scheduler started.")
}