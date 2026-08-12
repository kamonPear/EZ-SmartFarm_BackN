package database

import (
	"time"
	"EZ-SmartFarm_BachN/models"
	"gorm.io/gorm"
)

func SaveCoopLayout(coopID int, slots []models.SlotPayload) error {
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

func GetAllDevices() ([]models.Device, error) {
	var devices []models.Device
	err := DB.Find(&devices).Error
	return devices, err
}

func GetDeviceByID(id int) (*models.Device, error) {
	var device models.Device
	err := DB.First(&device, id).Error
	return &device, err
}

func GetDevicesByCoopID(coopID int) ([]models.Device, error) {
	var devices []models.Device
	err := DB.Where("coop_id = ?", coopID).Find(&devices).Error
	return devices, err
}

func CreateDevice(device *models.Device) error {
	device.LastUpdate = time.Now()
	if device.CurrentStatus == "" {
		device.CurrentStatus = "Offline"
	}
	return DB.Create(device).Error
}

func UpdateDevice(device *models.Device) error {
	device.LastUpdate = time.Now()
	return DB.Model(device).Updates(device).Error
}

func DeleteDevice(id int) error {
	return DB.Delete(&models.Device{}, id).Error
}