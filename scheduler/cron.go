package scheduler

import (
	"log"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/handlers"
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
		log.Println("⏰ [Cron] กำลังตัดสต็อกอาหารประจำวันตามจำนวนไก่จริงของแต่ละผู้ใช้...")

		// เดิมตัดเท่ากันทุกบัญชี (เม็ดเล็ก 20 kg, เม็ดใหญ่ 30 kg) ไม่ว่าใครจะเลี้ยง
		// ไก่กี่ตัวก็ตาม - ตอนนี้คำนวณแยกต่อผู้ใช้จากจำนวนไก่จริงในแต่ละคอกของเขา
		// (ComputeDailyFoodConsumption ตัวเดียวกับที่ ForceDeductStockHandler และ
		// หน้าคลังอาหารใช้ ให้ตัวเลขตรงกันทุกจุด) ดึงคอกทั้งระบบมาครั้งเดียวแล้ว
		// ค่อยแบ่งกลุ่มตามเจ้าของ แทนที่จะวนคิวรีทีละบัญชี
		coopsByUser, err := database.GetAllCoopsGroupedByUser()
		if err != nil {
			log.Printf("❌ [Cron] ดึงข้อมูลคอกไก่ทั้งหมดล้มเหลว: %v\n", err)
			return
		}

		deductedUsers := 0
		for userID, coops := range coopsByUser {
			userDeducted := false
			for _, foodType := range []string{models.FoodTypeSmallPellet, models.FoodTypeLargePellet} {
				items := handlers.ComputeDailyFoodConsumptionByCoop(coops, foodType)
				var amount float64
				for _, item := range items {
					amount += item.KgGiven
				}
				if amount <= 0 {
					continue // ไม่มีคอกไหนของผู้ใช้นี้กำลังกินอาหารประเภทนี้อยู่
				}
				if err := database.DeductFoodstockByType(foodType, userID, amount); err != nil {
					log.Printf("❌ [Cron] ตัดสต็อก %s ของผู้ใช้ %d ล้มเหลว: %v\n", foodType, userID, err)
					continue
				}
				// บันทึก breakdown รายคอกไว้ด้วย ให้หน้า "สรุปผลอาหารแต่ละประเภท" เห็น
				// ว่าคอกไหนกินไปเท่าไรได้ แม้ตัดโดย Cron ไม่มีใครกรอกเอง
				if _, err := database.CreateFoodDistributionBatch(foodType, items, userID); err != nil {
					log.Printf("❌ [Cron] บันทึก breakdown รายคอกของผู้ใช้ %d ล้มเหลว (ตัดสต็อกไปแล้ว): %v\n", userID, err)
				}
				userDeducted = true
			}
			if userDeducted {
				deductedUsers++
			}
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