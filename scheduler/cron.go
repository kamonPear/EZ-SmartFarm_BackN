package scheduler

import (
	"log"

	"EZ-SmartFarm_BachN/handlers"
	"EZ-SmartFarm_BachN/models"
	"github.com/robfig/cron/v3"
)

// SetupJobs สำหรับตั้งเวลาการทำงานอัตโนมัติ
func SetupJobs() {
	// สร้าง cron instance - ต้องระบุ location เป็นเวลาไทยตรงๆ (models.BangkokLocation)
	// ไม่งั้น cron จะยึดตามเวลา local ของ "เครื่อง/container ที่รันโปรเซส" ซึ่งบน
	// Render คือ UTC ไม่ใช่เวลาไทย ทำให้ "0 6 * * *" ที่ตั้งใจไว้ว่าคือ 6 โมงเช้าไทย
	// กลายเป็นทำงานจริงตอนบ่าย 1 (06:00 UTC = 13:00 ICT) แทน
	c := cron.New(cron.WithLocation(models.BangkokLocation()))

	// ตั้งเวลา: ทำงานทุกวัน เวลา 06:00 น. (6 โมงเช้า)
	// เปลี่ยนเป็น "* * * * *" ถ้าต้องการทดสอบให้ทำงานทุกๆ 1 นาที
	_, err := c.AddFunc("0 6 * * *", func() {
		log.Println("⏰ [Cron] กำลังตัดสต็อกอาหารประจำวันตามจำนวนไก่จริงของแต่ละผู้ใช้...")

		// ใช้ฟังก์ชันกลางร่วมกับ RunScheduledFoodDeductionHandler (ตัวปลุกงานจาก
		// ภายนอก) กันไม่ให้ลอจิก "ตัดสต็อกให้ทุกผู้ใช้" ซ้ำอยู่ 2 ที่
		deductedUsers, err := handlers.RunDailyDeductionForAllUsers()
		if err != nil {
			log.Printf("❌ [Cron] ตัดสต็อกอาหารประจำวันล้มเหลว: %v\n", err)
			return
		}

		log.Printf("✅ [Cron] ตัดสต็อกอาหารประจำวันเสร็จสิ้น (%d ผู้ใช้)\n", deductedUsers)
	})

	if err != nil {
		log.Fatalf("ตั้งค่า Cron Job ล้มเหลว: %v", err)
	}

	// เริ่มการทำงาน
	c.Start()
	log.Println("✅ [Cron] Scheduler started.")
}