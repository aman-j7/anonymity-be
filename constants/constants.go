package constants

import "time"

const (
	ActivePlayerCount = 3
	CleanupInterval   = 60 * time.Second
	MaxIdleTime       = 5 * time.Minute
	MaxRoomCount      = 100
	RoomBatchSize     = 25
)
