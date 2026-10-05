package services

import (
	"encoding/json"
	"fmt"
	"time"

	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// thresholdPayload คือข้อความที่ Arduino รับจาก topic farm/config/<coop_id>/threshold
type thresholdPayload struct {
	Temperature float64 `json:"temperature"`
	Ammonia     float64 `json:"ammonia"`
}

func thresholdTopic(coopID int32) string {
	return fmt.Sprintf("farm/config/%d/threshold", coopID)
}

// publishThresholdToCoop ส่งค่ามาตรฐานไปที่คอกเดียวแบบ retained เพื่อให้ Arduino ที่เพิ่ง
// เปิดเครื่องหรือเพิ่ง reconnect ได้ค่าล่าสุดทันทีโดยไม่ต้องรอให้ใครกดบันทึกใหม่
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

// PublishThresholdsForUser ส่งค่ามาตรฐานของ userID ไปให้ทุกคอกของผู้ใช้คนนั้น
// เรียกหลังบันทึกค่ามาตรฐานใหม่ (PUT /api/farm-threshold)
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

// PublishAllThresholds ส่งค่ามาตรฐานให้ทุกคอกของทุกผู้ใช้ เรียกตอน MQTT เชื่อมต่อ/เชื่อมต่อใหม่
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
