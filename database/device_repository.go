package database

import (
	"time"

	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SaveCoopLayout บันทึกผังอุปกรณ์ของคอก (ต้องเป็นคอกของ userID) แบบ "ซิงก์":
//   - อุปกรณ์ที่หน้าเว็บส่ง slot_index (เลขที่ backend เคยออกให้) มาด้วย = ตัวเดิม คงเลขเดิมและประวัติค่าเซนเซอร์ไว้
//     แล้วอัปเดตแค่ชื่อ/ไอคอน/ตำแหน่ง
//   - อุปกรณ์ที่ไม่ส่งเลขมา = ตัวใหม่ ได้เลขถัดไปของคอกนี้ (เริ่ม 1 ต่อคอก วิ่งขึ้นไม่ซ้ำ)
//   - อุปกรณ์เดิมที่หายไปจากรายการ = ถูกเอาออก (ลบพร้อมประวัติค่าเซนเซอร์ของมัน)
// slot.ID (เลขช่องกริดบนหน้าเว็บ) ไม่ถูกเก็บแล้ว ตำแหน่งจริงเก็บที่ pos_x/pos_y
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

// GetAllDevices retrieves every device belonging to any of userID's coops
func GetAllDevices(userID int) ([]models.Device, error) {
	var devices []models.Device
	err := DB.Where("coop_id IN (SELECT coop_id FROM coop WHERE user_id = ?)", userID).Find(&devices).Error
	return devices, err
}

// GetDeviceByID retrieves a device by ID, only if its coop belongs to userID
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

// GetDevicesByCoopID retrieves every device for coopID, only if it belongs to userID
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

// CreateDevice creates a new device, only if device.CoopID belongs to userID
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

// UpdateDevice updates an existing device, only if it (and any target coop) belongs to userID
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

// DeleteDevice deletes a device by ID, only if its coop belongs to userID
func DeleteDevice(id int, userID int) error {
	device, err := GetDeviceByID(id, userID)
	if err != nil {
		return err
	}
	return DB.Delete(device).Error
}

// DeleteDeviceBySlot deletes the device at (coopID, slotIndex), only if coopID belongs to userID
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
