// =====================================================================================
// ไฟล์ services/slot_publisher.go - ส่ง "เลขช่องของอุปกรณ์แต่ละชนิด" ไปให้บอร์ดของแต่ละคอก
// =====================================================================================
//
// ปัญหาที่แก้: ผู้ใช้จัดวางอุปกรณ์ในคอกผ่านหน้าเว็บ แต่ละตัวได้เลขช่อง (slot_index)
// บอร์ดต้องรู้ว่า DHT22 อยู่ช่องไหน พัดลมอยู่ช่องไหน เพื่อส่งค่าพร้อมเลขช่องที่ถูกต้อง
// แทนที่จะฝังเลขไว้ในโค้ดบอร์ด (ต้องแฟลชใหม่ทุกครั้งที่ย้ายอุปกรณ์) backend จึงส่งเลขให้ทาง MQTT
// topic: farm/config/<coop_id>/slots ตัวอย่างข้อความ {"dht":4, "mq135":9, "fans":[2,5]}
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

// slotsTopic สร้างชื่อ topic ของคอก เช่น คอก 3 -> "farm/config/3/slots"
// แยก topic ต่อคอก บอร์ดแต่ละตัวจะ subscribe เฉพาะคอกของตัวเอง
func slotsTopic(coopID int32) string {
	return fmt.Sprintf("farm/config/%d/slots", coopID)
}

// slotKeyForName แปลงชื่ออุปกรณ์ที่ผู้ใช้เห็นเป็นคีย์สั้นที่โค้ดบอร์ดรู้จัก
//  DHT22 -> "dht", MQ-135 -> "mq135", PIR MOTION -> "pir", พัดลม -> "fan", หลอดไฟ -> "light"
// ก่อนเทียบจะตัดช่องว่าง/ขีด และทำเป็นตัวพิมพ์ใหญ่ เพื่อให้ "MQ 135", "mq-135" ได้ผลเหมือนกัน
// ชนิดที่ไม่รู้จัก: สร้างคีย์จากชื่อเอง (เหลือ a-z0-9 เช่น "MC-38" -> "mc38")
// เพิ่มเซนเซอร์ใหม่จึงแก้แค่โค้ดบอร์ด ไม่ต้องแก้ backend ชื่อไทยที่ไม่รู้จักจะได้ "" (ข้ามไป)
// ถูกเรียกจาก: BuildSlotMap
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

// BuildSlotMap สรุปอุปกรณ์ของคอกเป็น {คีย์ชนิด: เลขช่อง}
//  - เซนเซอร์ (dht, mq135, pir ฯลฯ): ถ้ามีชนิดเดียวกันหลายตัว ใช้ตัวที่เลขช่องน้อยสุด
//    (บอร์ดอ่านเซนเซอร์แต่ละชนิดได้ทีละตัว)
//  - พัดลม/หลอดไฟ: มีได้หลายตัว ส่งเป็น array เรียงเลขจากน้อยไปมาก เช่น "fans": [2, 5]
//    ลำดับนี้ใช้จับคู่กับขารีเลย์บนบอร์ดตามลำดับ
//    และยังส่ง "fan"/"light" (ตัวเลขน้อยสุด) ไว้ให้เฟิร์มแวร์รุ่นเก่าที่รู้จักแค่ตัวเดียว
// แยกเป็นฟังก์ชันที่ไม่แตะ DB/MQTT เพื่อให้เขียน unit test ได้ (slot_publisher_test.go)
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

// PublishSlotsForCoop ดึงอุปกรณ์ของคอกจาก DB -> สร้าง slot map -> publish ไปที่ topic ของคอก
// ส่งแบบ retained = broker จำข้อความล่าสุดไว้ บอร์ดที่เพิ่งเปิดหรือเพิ่งต่อใหม่ได้ค่าทันที
// QoS 1 = รับประกันว่าถึงอย่างน้อย 1 ครั้ง รอผลไม่เกิน 5 วินาที
// ส่ง {} ได้ถ้าคอกไม่มีอุปกรณ์ที่บอร์ดรู้จักเหลือ บอร์ดจะได้หยุดส่งค่าของช่องที่ถูกเอาออก
// ถ้า MQTT ยังไม่เชื่อมต่อ ข้ามไป (ตอนต่อสำเร็จจะส่งให้ทุกคอกอยู่แล้ว)
// ถูกเรียกจาก: handlers/device.go (หลังบันทึกผัง / ลบอุปกรณ์) และ handlers/data_purge.go
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

// PublishAllSlots ส่งเลขช่องให้ทุกคอกในระบบ (ดึง coop_id ทั้งหมดแล้วเรียก PublishSlotsForCoop ทีละคอก)
// ถูกเรียกจาก: mqtt_service.go ตอนเชื่อมต่อ / เชื่อมต่อใหม่สำเร็จ
// กันกรณี broker รีสตาร์ตแล้วข้อความ retained เดิมหายไป
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
