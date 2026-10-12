package services

import (
	"testing"

	"EZ-SmartFarm_BachN/models"
)

func TestBuildSlotMap(t *testing.T) {
	devices := []models.Device{
		{Name: "DHT22", SlotIndex: 7},
		{Name: "DHT22", SlotIndex: 4}, // ชนิดซ้ำ - ใช้ช่องน้อยสุด
		{Name: "MQ-135", SlotIndex: 9},
		{Name: "PIR MOTION", SlotIndex: 13},
		{Name: "พัดลม", SlotIndex: 2},
		{Name: "หลอดไฟ", SlotIndex: 17},
		{Name: "MC-38", SlotIndex: 5}, // บอร์ดยังไม่รองรับ - ต้องไม่อยู่ในผลลัพธ์
	}
	got := BuildSlotMap(devices)
	want := map[string]int32{"dht": 4, "mq135": 9, "pir": 13, "fan": 2, "light": 17}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %d, want %d", k, got[k], v)
		}
	}
	if len(BuildSlotMap(nil)) != 0 {
		t.Error("empty coop should give an empty map")
	}
}
