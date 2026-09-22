package database

import (
	"time"

	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"
)

// SaveCoopLayout replaces every device slot for coopID, only if coopID belongs to userID
func SaveCoopLayout(coopID int, slots []models.SlotPayload, userID int) error {
	owned, err := CoopBelongsToUser(coopID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrNotFound
	}

	return DB.Transaction(func(tx *gorm.DB) error {

		if err := tx.Exec("DELETE FROM sensor_log WHERE device_id IN (SELECT device_id FROM device WHERE coop_id = ?)", coopID).Error; err != nil {
			return err
		}

		if err := tx.Where("coop_id = ?", coopID).Delete(&models.Device{}).Error; err != nil {
			return err
		}

		for _, slot := range slots {
			if slot.Device != nil {
				deviceType := "Sensor"
				if slot.Device.Name == "พัดลม" || slot.Device.Name == "หลอดไฟ" {
					deviceType = "Actuator"
				}

				newDevice := models.Device{
					CoopID:        int32(coopID),
					SlotIndex:     int32(slot.ID),
					Name:          slot.Device.Name,
					Icon:          slot.Device.Icon,
					DeviceType:    deviceType,
					CurrentStatus: "Offline",
					LastUpdate:    time.Now(),
				}

				if err := tx.Create(&newDevice).Error; err != nil {
					return err
				}
			}
		}

		return nil
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
	return DB.Create(device).Error
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
