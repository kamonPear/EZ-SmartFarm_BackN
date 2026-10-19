// =====================================================================================
// ไฟล์ database/ownership.go - ตัวเช็กความเป็นเจ้าของคอก (ใช้ร่วมกันหลายตาราง)
// =====================================================================================
//
// ข้อมูลอย่างไข่ สุขภาพ ประวัติวัคซีน อุปกรณ์ ไม่มี user_id ของตัวเอง แต่ผูกกับคอก (coop_id)
// และคอกผูกกับผู้ใช้ (user_id) อีกที การเช็กสิทธิ์จึงทำผ่าน "คอกนี้เป็นของผู้ใช้ไหม"
package database

import "EZ-SmartFarm_BachN/models"

// CoopBelongsToUser ตอบว่าคอก coopID มีอยู่จริงและเป็นของผู้ใช้ userID หรือไม่
// ทำงาน: นับแถวใน coop ที่ coop_id และ user_id ตรงกัน ถ้านับได้ > 0 = เป็นเจ้าของ
// ใช้ Count แทนการดึงทั้งแถว เพราะต้องการแค่ "มี/ไม่มี" เร็วและเบากว่า
// ทำไมแยกเป็นฟังก์ชันกลาง: repository ของไข่ สุขภาพ วัคซีน อุปกรณ์ จะได้ไม่ต้องเขียน query ซ้ำกันทุกไฟล์
// ถูกเรียกจาก: egg/health/vaccine/device/coop/sensor_log repository และ handlers หลายตัว
func CoopBelongsToUser(coopID, userID int) (bool, error) {
	var count int64
	if err := DB.Model(&models.Coop{}).Where("coop_id = ? AND user_id = ?", coopID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
