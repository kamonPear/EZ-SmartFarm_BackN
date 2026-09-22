package database

import "EZ-SmartFarm_BachN/models"

// CoopBelongsToUser reports whether coopID exists and is owned by userID. Shared by
// every coop-scoped entity repository (egg, health, vaccine, device) so they don't
// each repeat the same existence+ownership query.
func CoopBelongsToUser(coopID, userID int) (bool, error) {
	var count int64
	if err := DB.Model(&models.Coop{}).Where("coop_id = ? AND user_id = ?", coopID, userID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
