package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// slotsTopic คือ topic ที่ Arduino ของแต่ละคอกรับ "เลขช่อง (slot_index) ของอุปกรณ์แต่ละชนิด"
// ที่ผู้ใช้วางไว้ในคอกนั้นบนหน้าเว็บ - บอร์ดจะได้ไม่ต้องฝังเลขช่องไว้ในโค้ด
func slotsTopic(coopID int32) string {
	return fmt.Sprintf("farm/config/%d/slots", coopID)
}

// slotKeyForName แปลงชื่อชนิดอุปกรณ์ที่ผู้ใช้เห็น (เช่น "DHT22", "MQ-135", "PIR MOTION",
// "พัดลม", "หลอดไฟ") เป็นคีย์สั้นคงที่ที่โค้ดบนบอร์ดรู้จัก คืน "" ถ้าไม่รู้จัก (เช่น MC-38
// ที่บอร์ดยังไม่รองรับ) - เทียบแบบไม่สนตัวพิมพ์ ช่องว่าง และขีด
func slotKeyForName(name string) string {
	n := strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "_", "").Replace(name))
	switch {
	case strings.HasPrefix(n, "DHT"):
		return "dht"
	case strings.HasPrefix(n, "MQ135"):
		return "mq135"
	case strings.HasPrefix(n, "PIR"):
		return "pir"
	case n == "พัดลม":
		return "fan"
	case n == "หลอดไฟ":
		return "light"
	}
	return ""
}

// BuildSlotMap สรุปอุปกรณ์ของคอกเป็น {คีย์ชนิด: slot_index} ถ้าคอกมีชนิดเดียวกันหลายตัว
// ใช้ตัวที่เลขช่องน้อยที่สุด (บอร์ดหนึ่งตัวรองรับเซนเซอร์ชนิดละหนึ่งตัว)
func BuildSlotMap(devices []models.Device) map[string]int32 {
	slots := make(map[string]int32)
	for _, d := range devices {
		key := slotKeyForName(d.Name)
		if key == "" {
			continue
		}
		if existing, ok := slots[key]; !ok || d.SlotIndex < existing {
			slots[key] = d.SlotIndex
		}
	}
	return slots
}

// PublishSlotsForCoop ส่งเลขช่องของคอกนี้ไปที่ MQTT แบบ retained (บอร์ดที่เพิ่งเปิดหรือ
// เพิ่ง reconnect ได้ค่าล่าสุดทันที) - เรียกหลังบันทึกผัง/ลบอุปกรณ์ ส่ง {} ได้ถ้าคอกไม่มี
// อุปกรณ์ที่บอร์ดรู้จักเหลือเลย เพื่อให้บอร์ดหยุดส่งข้อมูลของช่องที่ถูกเอาออกแล้ว
func PublishSlotsForCoop(coopID int32) {
	if mqttClient == nil || !mqttClient.IsConnected() {
		fmt.Printf("⚠️ [MQTT] ยังไม่ได้เชื่อมต่อ broker ข้ามการส่งเลขช่องของคอก %d\n", coopID)
		return
	}
	db := database.GetDB()
	if db == nil {
		return
	}

	var devices []models.Device
	if err := db.Where("coop_id = ?", coopID).Find(&devices).Error; err != nil {
		fmt.Printf("❌ [DB Error] ดึงอุปกรณ์ของคอก %d ไม่ได้: %v\n", coopID, err)
		return
	}

	body, err := json.Marshal(BuildSlotMap(devices))
	if err != nil {
		return
	}

	token := mqttClient.Publish(slotsTopic(coopID), 1, true, body)
	if !token.WaitTimeout(5 * time.Second) {
		fmt.Printf("⚠️ [MQTT] ส่งเลขช่องของคอก %d ไม่ทันเวลา\n", coopID)
		return
	}
	if token.Error() != nil {
		fmt.Printf("❌ [MQTT Publish Error] เลขช่องคอก %d: %v\n", coopID, token.Error())
		return
	}
	fmt.Printf("📤 [MQTT] ส่งเลขช่องคอก %d -> %s\n", coopID, string(body))
}

// PublishAllSlots ส่งเลขช่องให้ทุกคอก เรียกตอน MQTT เชื่อมต่อ/เชื่อมต่อใหม่
func PublishAllSlots() {
	db := database.GetDB()
	if db == nil {
		return
	}
	var coopIDs []int32
	if err := db.Model(&models.Coop{}).Pluck("coop_id", &coopIDs).Error; err != nil {
		fmt.Println("❌ [DB Error] ดึงรายการคอกไม่ได้:", err)
		return
	}
	for _, id := range coopIDs {
		PublishSlotsForCoop(id)
	}
}
