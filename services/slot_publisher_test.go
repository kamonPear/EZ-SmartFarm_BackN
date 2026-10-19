// =====================================================================================
// ไฟล์ services/slot_publisher_test.go - unit test ของ BuildSlotMap (รันด้วยคำสั่ง go test ./services)
// =====================================================================================
package services

import (
	"encoding/json"
	"reflect"
	"testing"

	"EZ-SmartFarm_BachN/models"
)

// TestBuildSlotMap ทดสอบว่า BuildSlotMap แปลงรายการอุปกรณ์เป็น slot map ถูกต้องทุกกรณี:
//  - เซนเซอร์ชนิดซ้ำ (DHT22 สองตัว) ต้องได้ช่องที่น้อยสุด
//  - พัดลม/หลอดไฟหลายตัว ต้องได้ array เรียงจากน้อยไปมาก และคีย์เดี่ยว fan/light ด้วย
//  - ชนิดใหม่ (MC-38) ต้องได้คีย์จากชื่อ "mc38"
//  - ชื่อไทยที่ไม่รู้จัก ต้องไม่อยู่ในผลลัพธ์
//  - ผลลัพธ์แปลงเป็น JSON ได้ และคอกที่ไม่มีอุปกรณ์ได้ map ว่าง
func TestBuildSlotMap(t *testing.T) {
	devices := []models.Device{
		{Name: "DHT22", SlotIndex: 7},
		{Name: "DHT22", SlotIndex: 4}, // เซนเซอร์ชนิดซ้ำ - ใช้ช่องน้อยสุด
		{Name: "MQ-135", SlotIndex: 9},
		{Name: "PIR MOTION", SlotIndex: 13},
		{Name: "พัดลม", SlotIndex: 5},
		{Name: "พัดลม", SlotIndex: 2}, // พัดลมหลายตัว - ส่งเป็น array เรียงจากน้อยไปมาก
		{Name: "หลอดไฟ", SlotIndex: 18},
		{Name: "หลอดไฟ", SlotIndex: 17},
		{Name: "MC-38", SlotIndex: 5},        // ชนิดใหม่ - ส่งต่อด้วยคีย์จากชื่อ บอร์ดที่ไม่รู้จักจะข้ามเอง
		{Name: "เซนเซอร์ใหม่", SlotIndex: 6}, // ชื่อไทยที่ไม่รู้จัก - สร้างคีย์ไม่ได้ ต้องไม่อยู่ในผลลัพธ์
	}
	got := BuildSlotMap(devices)
	want := map[string]any{
		"dht": int32(4), "mq135": int32(9), "pir": int32(13), "mc38": int32(5),
		"fan": int32(2), "fans": []int32{2, 5},
		"light": int32(17), "lights": []int32{17, 18},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("slot map should marshal to valid JSON: %v", err)
	}

	if len(BuildSlotMap(nil)) != 0 {
		t.Error("empty coop should give an empty map")
	}
}
