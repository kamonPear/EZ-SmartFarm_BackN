// =====================================================================================
// ไฟล์ handlers/food_age_rule.go - API ตารางปริมาณอาหารตามอายุไก่
// =====================================================================================
//
// หน้าเว็บคลังอาหารมีหน้าต่างให้ผู้ใช้ตั้งว่า ไก่อายุช่วงไหน กินอาหารอะไร ตัวละกี่กรัมต่อวัน
// ข้อมูลอยู่ใน database/food_age_rule_repository.go
// ตารางนี้ถูกใช้คำนวณการตัดสต็อกอาหารประจำวัน (handlers/foodstock.go)
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"EZ-SmartFarm_BachN/auth"
	"EZ-SmartFarm_BachN/database"
	"EZ-SmartFarm_BachN/models"
)

// GetFoodAgeRulesHandler (GET /api/food-age-rules) ส่งตารางให้อาหารของผู้ใช้
// ถ้ายังไม่เคยตั้ง จะได้ list ว่าง [] (หน้าเว็บจะให้ผู้ใช้เริ่มกรอก)
func GetFoodAgeRulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rules, err := database.GetFoodAgeRules(userID)
	if err != nil {
		http.Error(w, "Failed to fetch food age rules", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// UpdateFoodAgeRulesHandler (PUT /api/food-age-rules) บันทึกตารางให้อาหารใหม่ทั้งตาราง
// body: {"rules":[{"food_type":"เม็ดเล็ก","min_weeks":0,"max_weeks":14,"grams_per_bird":60}, ...]}
//  1. ตรวจข้อมูลด้วย req.Validate() (models/food_age_rule.go)
//     เช่น ประเภทอาหารต้องถูกต้อง, ช่วงอายุไม่ติดลบ/ไม่ทับกัน, กรัมต้องมากกว่า 0
//     ไม่ผ่าน -> 400 พร้อมข้อความบอกว่าผิดตรงไหน
//  2. แทนที่ทั้งตารางด้วย database.ReplaceFoodAgeRules (ใน Transaction)
//  3. ส่งตารางใหม่กลับไปให้หน้าเว็บแสดง
func UpdateFoodAgeRulesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := auth.UserIDFromContext(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req models.UpdateFoodAgeRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if msg := req.Validate(); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	rules, err := database.ReplaceFoodAgeRules(userID, req.Rules)
	if err != nil {
		http.Error(w, "Failed to save food age rules", http.StatusInternalServerError)
		return
	}
	log.Printf("[%s] %s - %d ✓ Saved %d food age rules for user %d", r.Method, r.RequestURI, http.StatusOK, len(rules), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}
