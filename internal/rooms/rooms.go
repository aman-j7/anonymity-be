package rooms

import (
	"anonymity/constants"
	"anonymity/internal/infra"
	"context"
	"log"
	"math/rand"
	"time"
)

const (
	codeChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength = 6
	maxRetries = 5
	roomTTL    = 1 * time.Hour
)

func GenerateRoomCode(ctx context.Context) string {
	code := randomCode()
	_, err := infra.Redis.SAdd(ctx, constants.RoomCodeContainerKey, code).Result()
	if err != nil {
		log.Printf("Unable to push code in container : %v", err)
	}
	return code
}

func randomCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = codeChars[rand.Intn(len(codeChars))]
	}
	return string(b)
}

func RoomCount() int {
	ctx := context.Background()
	count, roomCodesError := infra.Redis.SCard(ctx, constants.RoomCodeContainerKey).Result()

	if roomCodesError != nil {
		log.Printf("Error occurred fetching room count: %v", roomCodesError)
		return 0
	}

	return int(count)
}
