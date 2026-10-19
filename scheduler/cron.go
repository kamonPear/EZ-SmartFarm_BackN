// =====================================================================================
// ไฟล์ scheduler/cron.go - ตั้งเวลางานอัตโนมัติ (ตัดสต็อกอาหารทุกวัน 18:00 น.)
// =====================================================================================
//
// cron = ตัวตั้งเวลาให้ฟังก์ชันทำงานซ้ำตามเวลาที่กำหนด (ใช้ไลบรารี robfig/cron)
// รูปแบบ "นาที ชั่วโมง วัน เดือน วันในสัปดาห์" เช่น "0 18 * * *" = ทุกวัน 18:00 น.
package scheduler

import (
	"fmt"
	"log"

	"EZ-SmartFarm_BachN/handlers"
	"EZ-SmartFarm_BachN/models"
	"github.com/robfig/cron/v3"
)

// SetupJobs สร้าง cron และลงทะเบียนงาน "ตัดสต็อกอาหารประจำวัน" เวลา 18:00 น.
//
// ทำงานอย่างไร:
//  1. สร้าง cron โดยระบุเขตเวลาไทย (models.BangkokLocation)
//     ทำไม: เซิร์ฟเวอร์บน Render ใช้เวลา UTC ถ้าไม่ระบุ "18:00" จะกลายเป็นตี 1 เวลาไทย
//  2. ทุกวัน 18:00 น. (handlers.DailyDeductionHour) เรียก handlers.RunDailyDeductionForAllUsers
//     = คำนวณอาหารที่ไก่ทุกคอกกินต่อวัน (ตามอายุไก่และตารางให้อาหาร) แล้วตัดสต็อกของทุกผู้ใช้
//     ใช้ฟังก์ชันเดียวกับตัวปลุกงานจากภายนอก (/api/tasks/...) เพื่อไม่ให้ลอจิกซ้ำกัน 2 ที่
//  3. c.Start() เริ่มนับเวลา (cron ทำงานอยู่เบื้องหลัง ไม่บล็อก main)
// ถ้าตั้งเวลาผิดรูปแบบ จะหยุดโปรแกรม (log.Fatalf) เพราะเป็นความผิดพลาดของโค้ด ควรรู้ทันที
// ถูกเรียกจาก: main.go
func SetupJobs() {
	// สร้าง cron instance - ต้องระบุ location เป็นเวลาไทยตรงๆ (models.BangkokLocation)
	// ไม่งั้น cron จะยึดตามเวลา local ของ "เครื่อง/container ที่รันโปรเซส" ซึ่งบน
	// Render คือ UTC ไม่ใช่เวลาไทย ทำให้ "0 18 * * *" ที่ตั้งใจไว้ว่าคือ 6 โมงเย็นไทย
	// กลายเป็นทำงานจริงตอนตี 1 (18:00 UTC = 01:00 ICT) แทน
	c := cron.New(cron.WithLocation(models.BangkokLocation()))

	// ตั้งเวลา: ทำงานทุกวัน เวลา 18:00 น. (6 โมงเย็น) ตาม handlers.DailyDeductionHour
	_, err := c.AddFunc(fmt.Sprintf("0 %d * * *", handlers.DailyDeductionHour), func() {
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
