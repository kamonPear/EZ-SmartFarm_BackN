package models

// UpdateFarmLayoutRequest represents the request payload for changing the farm's chosen shape
type UpdateFarmLayoutRequest struct {
	Shape string `json:"shape" binding:"required"`
}

// CoopPositionEntry represents one coop's position on the farm layout canvas.
// PosX/PosY are nil to unplace a coop (send it back to the "not yet placed" tray).
type CoopPositionEntry struct {
	CoopID int      `json:"coop_id"`
	PosX   *float64 `json:"pos_x"`
	PosY   *float64 `json:"pos_y"`
}

// UpdateCoopPositionsRequest represents the request payload for saving every coop's
// position on the farm layout canvas in one batch (mirrors how the device layout page
// saves all 21 slots at once).
type UpdateCoopPositionsRequest struct {
	Positions []CoopPositionEntry `json:"positions"`
}
