package scheduler

import (
	"log"

	"EZ-SmartFarm_BachN/database"
	"github.com/robfig/cron/v3"
)

// SetupJobs สำหรับตั้งเวลาการทำงานอัตโนมัติ
func SetupJobs() {
	// สร้าง cron instance
	c := cron.New()

	// ตั้งเวลา: ทำงานทุกวัน เวลา 00:00 น. (เที่ยงคืน)
	// เปลี่ยนเป็น "* * * * *" ถ้าต้องการทดสอบให้ทำงานทุกๆ 1 นาที
	_, err := c.AddFunc("0 0 * * *", func() {
		log.Println("⏰ [Cron] กำลังตัดสต็อกอาหารประจำวัน (20 kg)...")
		
		// เรียกใช้ฟังก์ชันจาก database หักออก 20 กก.
		err := database.DeductDailyFoodstock(20.0) 
		if err != nil {
			log.Printf("❌ [Cron] ตัดสต็อกล้มเหลว: %v\n", err)
		}
	})

	if err != nil {
		log.Fatalf("ตั้งค่า Cron Job ล้มเหลว: %v", err)
	}

	// เริ่มการทำงาน
	c.Start()
	log.Println("✅ [Cron] Scheduler started.")
}