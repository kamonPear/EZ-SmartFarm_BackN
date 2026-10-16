package services

import (
	"encoding/json"
	"fmt"
	"sort"
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
// "พัดลม", "หลอดไฟ") เป็นคีย์สั้นคงที่ที่โค้ดบนบอร์ดรู้จัก ชนิดที่ไม่รู้จักใช้คีย์จากชื่อ
// (ดูท้ายฟังก์ชัน) - เทียบแบบไม่สนตัวพิมพ์ ช่องว่าง และขีด
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
	// ชนิดที่ยังไม่รู้จัก: ส่งต่อด้วยคีย์ที่ได้จากชื่อ (ตัวพิมพ์เล็ก เหลือเฉพาะ a-z0-9 เช่น "MC-38" -> "mc38")
	// เผื่อเซนเซอร์ใหม่ - เพิ่มเซนเซอร์แล้วแก้แค่โค้ดบอร์ดให้อ่านคีย์นี้ บอร์ดที่ยังไม่รู้จักคีย์จะข้ามไปเอง
	// ชื่อที่ไม่มีตัวอักษร/ตัวเลขภาษาอังกฤษเลย (เช่น ชื่อไทยที่ไม่รู้จัก) ยังคืน ""
	var key strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			key.WriteRune(r)
		}
	}
	return key.String()
}

// BuildSlotMap สรุปอุปกรณ์ของคอกเป็น {คีย์ชนิด: slot_index}
//   - เซนเซอร์ (dht, mq135, pir): ถ้าคอกมีชนิดเดียวกันหลายตัว ใช้ตัวที่เลขช่องน้อยที่สุด
//   - พัดลม/หลอดไฟ: คอกหนึ่งมีได้หลายตัว ส่งเป็น array เรียงเลขช่องจากน้อยไปมาก
//     ("fans": [1, 5], "lights": [17, 18]) ลำดับนี้ใช้จับคู่กับขารีเลย์ตามลำดับบนบอร์ด
//     และยังส่งคีย์เดิม "fan"/"light" (ตัวที่เลขน้อยสุด) ไว้ให้เฟิร์มแวร์รุ่นเก่า
func BuildSlotMap(devices []models.Device) map[string]any {
	single := make(map[string]int32)
	fans := []int32{}
	lights := []int32{}
	for _, d := range devices {
		switch key := slotKeyForName(d.Name); key {
		case "":
			continue
		case "fan":
			fans = append(fans, d.SlotIndex)
		case "light":
			lights = append(lights, d.SlotIndex)
		default:
			if existing, ok := single[key]; !ok || d.SlotIndex < existing {
				single[key] = d.SlotIndex
			}
		}
	}

	slots := make(map[string]any, len(single)+4)
	for k, v := range single {
		slots[k] = v
	}
	if len(fans) > 0 {
		sort.Slice(fans, func(i, j int) bool { return fans[i] < fans[j] })
		slots["fan"] = fans[0]
		slots["fans"] = fans
	}
	if len(lights) > 0 {
		sort.Slice(lights, func(i, j int) bool { return lights[i] < lights[j] })
		slots["light"] = lights[0]
		slots["lights"] = lights
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
