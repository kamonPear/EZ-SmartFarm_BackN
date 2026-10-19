// =====================================================================================
// ไฟล์ services/threshold_publisher.go - ส่ง "ค่ามาตรฐาน" อุณหภูมิ/แอมโมเนียไปให้บอร์ดของทุกคอก
// =====================================================================================
//
// ผู้ใช้ตั้งค่ามาตรฐานในหน้าเว็บ (ตาราง farm_threshold, 1 ค่าต่อผู้ใช้)
// บอร์ดรับค่านี้ไว้เป็นเกณฑ์ (ตาม models/farm_threshold.go ตอนนี้ยังเป็นค่าอ้างอิงที่แสดงบนหน้าจอ
// ระบบเปิด/ปิดพัดลม/หลอดไฟอัตโนมัติตามค่านี้ยังไม่ได้ทำ)
// topic: farm/config/<coop_id>/threshold ข้อความ {"temperature":32, "ammonia":25}
package services

import (
	"encoding/json"
	"fmt"
	"time"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// thresholdPayload รูปแบบ JSON ที่ส่งให้บอร์ด มีแค่ 2 ค่า: อุณหภูมิ (°C) และแอมโมเนีย (ppm)
type thresholdPayload struct {
	Temperature float64 `json:"temperature"`
	Ammonia     float64 `json:"ammonia"`
}

// thresholdTopic สร้างชื่อ topic ของคอก เช่น คอก 3 -> "farm/config/3/threshold"
func thresholdTopic(coopID int32) string {
	return fmt.Sprintf("farm/config/%d/threshold", coopID)
}

// publishThresholdToCoop ส่งค่ามาตรฐานไปที่คอกเดียว แบบ retained (QoS 1, รอไม่เกิน 5 วินาที)
// retained = broker จำค่าล่าสุดไว้ บอร์ดที่เพิ่งเปิดได้ค่าทันที ไม่ต้องรอให้ผู้ใช้กดบันทึกใหม่
// ถ้า MQTT ยังไม่เชื่อมต่อ ข้ามไป
// ถูกเรียกจาก: PublishThresholdsForUser
func publishThresholdToCoop(coopID int32, t *models.FarmThreshold) {
	if mqttClient == nil || !mqttClient.IsConnected() {
		fmt.Printf("⚠️ [MQTT] ยังไม่ได้เชื่อมต่อ broker ข้ามการส่งค่ามาตรฐานของคอก %d\n", coopID)
		return
	}

	body, err := json.Marshal(thresholdPayload{Temperature: t.Temperature, Ammonia: t.Ammonia})
	if err != nil {
		return
	}

	token := mqttClient.Publish(thresholdTopic(coopID), 1, true, body)
	if !token.WaitTimeout(5 * time.Second) {
		fmt.Printf("⚠️ [MQTT] ส่งค่ามาตรฐานของคอก %d ไม่ทันเวลา\n", coopID)
		return
	}
	if token.Error() != nil {
		fmt.Printf("❌ [MQTT Publish Error] คอก %d: %v\n", coopID, token.Error())
		return
	}
	fmt.Printf("📤 [MQTT] ส่งค่ามาตรฐานคอก %d -> %s\n", coopID, string(body))
}

// PublishThresholdsForUser ดึงค่ามาตรฐานของผู้ใช้ แล้วส่งให้ทุกคอกของผู้ใช้คนนั้น
// ทำไมส่งทุกคอก: ค่ามาตรฐานตั้งระดับฟาร์ม (ผู้ใช้) แต่บอร์ดติดอยู่รายคอก
// ถูกเรียกจาก: handlers/farm_threshold.go (หลังบันทึก PUT /api/farm-threshold) และ PublishAllThresholds
func PublishThresholdsForUser(userID int) {
	db := database.GetDB()
	if db == nil {
		return
	}

	threshold, err := database.GetFarmThreshold(userID)
	if err != nil {
		return
	}

	var coops []models.Coop
	if err := db.Where("user_id = ?", userID).Find(&coops).Error; err != nil {
		fmt.Println("❌ [DB Error] ดึงรายการคอกไม่ได้:", err)
		return
	}
	for _, coop := range coops {
		publishThresholdToCoop(int32(coop.CoopID), threshold)
	}
}

// PublishAllThresholds ส่งค่ามาตรฐานให้ทุกคอกของทุกผู้ใช้
// (ดึง user_id ที่มีคอกแบบไม่ซ้ำ แล้วเรียก PublishThresholdsForUser ทีละคน)
// ถูกเรียกจาก: mqtt_service.go ตอนเชื่อมต่อ / เชื่อมต่อใหม่สำเร็จ
func PublishAllThresholds() {
	db := database.GetDB()
	if db == nil {
		return
	}

	var userIDs []int
	if err := db.Model(&models.Coop{}).Distinct().Pluck("user_id", &userIDs).Error; err != nil {
		fmt.Println("❌ [DB Error] ดึงรายชื่อเจ้าของคอกไม่ได้:", err)
		return
	}
	for _, userID := range userIDs {
		PublishThresholdsForUser(userID)
	}
}
