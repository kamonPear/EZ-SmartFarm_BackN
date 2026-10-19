// =====================================================================================
// ไฟล์ database/device_repository.go - อุปกรณ์ที่ติดตั้งในคอก (ตาราง device)
// =====================================================================================
//
// อุปกรณ์ 1 ตัว = 1 แถว ผูกกับคอก (coop_id) มีชื่อ, ไอคอน, ประเภท (Sensor/Actuator),
// สถานะ (Online/Offline), ตำแหน่งบนผังคอก (pos_x, pos_y) และ "เลขอุปกรณ์ในคอก" (slot_index)
//
// เลขอุปกรณ์ (slot_index) เริ่มที่ 1 ในแต่ละคอก และไม่เปลี่ยนแม้บันทึกผังใหม่
// บอร์ด ESP32 ใช้เลขนี้บอกว่าค่าที่ส่งมาเป็นของอุปกรณ์ตัวไหน (ส่งเลขให้บอร์ดผ่าน MQTT: services/slot_publisher.go)
// ถูกเรียกจาก: handlers/device.go และ handlers/sensor_log.go
package database

import (
	"time"

	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SaveCoopLayout บันทึกผังอุปกรณ์ทั้งคอก (ปุ่ม "บันทึกการจัดวาง" ในหน้าจัดวางระบบเซนเซอร์บนเว็บ)
// ทำงานแบบ "ซิงก์" ให้ DB ตรงกับรายการที่หน้าเว็บส่งมา:
//   - อุปกรณ์ที่ส่งเลขเดิม (slot_index) มา = ตัวเดิม -> อัปเดตชื่อ/ไอคอน/ตำแหน่ง คงเลขและประวัติค่าไว้
//   - อุปกรณ์ที่ไม่มีเลข = ตัวใหม่ -> ออกเลขถัดไปของคอก (next_device_no)
//   - อุปกรณ์เดิมที่ไม่อยู่ในรายการ = ถูกเอาออก -> ลบพร้อมประวัติค่าเซนเซอร์
//
// รายละเอียด:
//  1. เช็กว่าคอกเป็นของผู้ใช้
//  2. เปิด Transaction และล็อกแถวคอก (SELECT ... FOR UPDATE)
//     กันกรณีกดบันทึกพร้อมกัน 2 ครั้งแล้วได้เลขอุปกรณ์ซ้ำ
//  3. ทำ map เลข -> อุปกรณ์เดิม และหาเลขถัดไปที่ยังไม่ถูกใช้
//  4. วนทุกช่องที่ส่งมา: ชื่อ "พัดลม" หรือ "หลอดไฟ" = Actuator (อุปกรณ์สั่งงาน) นอกนั้น = Sensor
//  5. ลบอุปกรณ์เดิมที่หายไป (ลบค่าใน sensor_log ก่อน เพราะเป็น Foreign Key)
//  6. บันทึกเลขถัดไปกลับลงคอก
//
// ถูกเรียกจาก: handlers/device.go (POST /api/coops/layout)
func SaveCoopLayout(coopID int, slots []models.SlotPayload, userID int) error {
	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		// ล็อกแถวคอกไว้ตลอดทรานแซกชัน กันสองคำขอบันทึกผังพร้อมกันได้เลขอุปกรณ์ซ้ำ
		var coop models.Coop
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("coop_id", "next_device_no").First(&coop, coopID).Error; err != nil {
			return err
		}

		var existing []models.Device
		if err := tx.Where("coop_id = ?", coopID).Find(&existing).Error; err != nil {
			return err
		}
		byNo := make(map[int32]models.Device, len(existing))
		next := int32(coop.NextDeviceNo)
		for _, d := range existing {
			byNo[d.SlotIndex] = d
			if d.SlotIndex >= next {
				next = d.SlotIndex + 1
			}
		}
		if next < 1 {
			next = 1
		}

		kept := make(map[int32]bool)
		for _, slot := range slots {
			if slot.Device == nil {
				continue
			}
			deviceType := "Sensor"
			if slot.Device.Name == "พัดลม" || slot.Device.Name == "หลอดไฟ" {
				deviceType = "Actuator"
			}

			// ตัวเดิม: มีเลขที่ตรงกับอุปกรณ์ที่มีอยู่ และยังไม่ถูกตัวอื่นในรายการเดียวกันอ้างไปก่อน
			if slot.Device.SlotIndex != nil {
				no := *slot.Device.SlotIndex
				if cur, ok := byNo[no]; ok && !kept[no] {
					updates := map[string]interface{}{
						"name":        slot.Device.Name,
						"icon":        slot.Device.Icon,
						"device_type": deviceType,
					}
					if slot.X != nil {
						updates["pos_x"] = *slot.X
					}
					if slot.Y != nil {
						updates["pos_y"] = *slot.Y
					}
					if err := tx.Model(&models.Device{}).Where("device_id = ?", cur.DeviceID).Updates(updates).Error; err != nil {
						return err
					}
					kept[no] = true
					continue
				}
			}

			// ตัวใหม่: ออกเลขถัดไปของคอกนี้
			newDevice := models.Device{
				CoopID:        int32(coopID),
				SlotIndex:     next,
				Name:          slot.Device.Name,
				Icon:          slot.Device.Icon,
				DeviceType:    deviceType,
				CurrentStatus: "Offline",
				LastUpdate:    time.Now(),
				PosX:          slot.X,
				PosY:          slot.Y,
			}
			if err := tx.Create(&newDevice).Error; err != nil {
				return err
			}
			next++
		}

		// ตัวเดิมที่หายไปจากรายการ = ถูกเอาออก (ลบประวัติค่าเซนเซอร์ของมันด้วย)
		for _, d := range existing {
			if kept[d.SlotIndex] {
				continue
			}
			if err := tx.Exec("DELETE FROM sensor_log WHERE device_id = ?", d.DeviceID).Error; err != nil {
				return err
			}
			if err := tx.Where("device_id = ?", d.DeviceID).Delete(&models.Device{}).Error; err != nil {
				return err
			}
		}

		return tx.Model(&models.Coop{}).Where("coop_id = ?", coopID).Update("next_device_no", next).Error
	})
}

// GetAllDevices ดึงอุปกรณ์ทุกตัวในทุกคอกของผู้ใช้ (หน้า "สถานะการทำงานของอุปกรณ์" / "อุปกรณ์รวม")
// ถูกเรียกจาก: handlers/device.go
func GetAllDevices(userID int) ([]models.Device, error) {
	var devices []models.Device
	err := DB.Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).Find(&devices).Error
	return devices, err
}

// GetDeviceByID ดึงอุปกรณ์ตัวเดียวตาม id เฉพาะในคอกของผู้ใช้ ไม่เจอคืน ErrNotFound
func GetDeviceByID(id int, userID int) (*models.Device, error) {
	var device models.Device
	err := DB.Where("device_id = ? AND coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", id, userID).First(&device).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &device, nil
}

// GetDevicesByCoopID ดึงอุปกรณ์ทั้งหมดในคอกเดียว หลังเช็กว่าคอกเป็นของผู้ใช้
// ใช้ในหน้าสถานะอุปกรณ์รายคอก และหน้าจัดวางเซนเซอร์ (โหลดผังเดิมมาแสดง)
func GetDevicesByCoopID(coopID int, userID int) ([]models.Device, error) {
	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, ErrNotFound
	}

	var devices []models.Device
	err = DB.Where("coop_id = ?", coopID).Find(&devices).Error
	return devices, err
}

// CreateDevice เพิ่มอุปกรณ์ทีละตัว (เช็กว่าคอกเป็นของผู้ใช้ก่อน)
// ตั้งสถานะเริ่มต้นเป็น Offline จนกว่าบอร์ดจะส่งข้อมูลมาครั้งแรก
// หลังบันทึก ดันตัวนับ next_device_no ของคอกให้ไม่ต่ำกว่าเลขที่เพิ่งใช้
// (GREATEST = เลือกค่าที่มากกว่า) กันการบันทึกผังครั้งถัดไปออกเลขซ้ำ
func CreateDevice(device *models.Device, userID int) error {
	owned, err := CoopBelongsToUser(int(device.CoopID), userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}

	device.LastUpdate = time.Now()
	if device.CurrentStatus == "" {
		device.CurrentStatus = "Offline"
	}
	if err := DB.Create(device).Error; err != nil {
		return err
	}
	// ตัวนับเลขอุปกรณ์ของคอกต้องไม่ต่ำกว่าเลขที่เพิ่งใช้ (ผู้เรียกกำหนดเลขเอง) ไม่งั้นการบันทึกผังครั้งต่อไปอาจออกเลขซ้ำ
	return DB.Exec("UPDATE coop SET next_device_no = GREATEST(next_device_no, ?) WHERE coop_id = ?", device.SlotIndex+1, device.CoopID).Error
}

// UpdateDevice แก้ไขข้อมูลอุปกรณ์ (เช็กว่าอุปกรณ์เป็นของผู้ใช้)
// ถ้าย้ายอุปกรณ์ไปคอกอื่น คอกปลายทางต้องเป็นของผู้ใช้ด้วย
// Updates(device) ของ GORM อัปเดตเฉพาะช่องที่ไม่ใช่ค่าศูนย์
func UpdateDevice(device *models.Device, userID int) error {
	existing, err := GetDeviceByID(int(device.DeviceID), userID)
	if err != nil {
		return err
	}

	// If the caller is moving the device to a different coop, that coop must be theirs too
	if device.CoopID != 0 && device.CoopID != existing.CoopID {
		owned, err := CoopBelongsToUser(int(device.CoopID), userID)
		if err != nil {
			return err
		}
		if !owned {
			return ErrNotFound
		}
	}

	device.LastUpdate = time.Now()
	return DB.Model(device).Updates(device).Error
}

// DeleteDevice ลบอุปกรณ์ตาม id (เช็กสิทธิ์ผ่าน GetDeviceByID)
func DeleteDevice(id int, userID int) error {
	device, err := GetDeviceByID(id, userID)
	if err != nil {
		return err
	}
	return DB.Delete(device).Error
}

// DeleteDeviceBySlot ลบอุปกรณ์ตามตำแหน่ง (คอก + เลขอุปกรณ์) หลังเช็กว่าคอกเป็นของผู้ใช้
func DeleteDeviceBySlot(coopID, slotIndex int, userID int) error {
	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}
	return DB.Where("coop_id = ? AND slot_index = ?", coopID, slotIndex).Delete(&models.Device{}).Error
}
