package utils

import (
	"context"
	"math/rand"
	"time"

	customerror "anonymity/internal/error"
	"anonymity/internal/infra"

	"github.com/redis/go-redis/v9"
)

const (
	codeChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength = 6
	maxRetries = 5
	roomTTL    = 2 * time.Hour
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GenerateRoomCode(ctx context.Context) (string, error) {
	for i := 0; i < maxRetries; i++ {
		code := randomCode()
		key := "room:" + code

		result, err := infra.Redis.SetArgs(ctx, key, "active", redis.SetArgs{
			Mode: "NX",
			TTL:  roomTTL,
		}).Result()

		if err != nil {
			return "", customerror.ServiceUnavailable("Redis error while generating room code")
		}

		if result == "OK" {
			return code, nil
		}
	}

	return "", customerror.ServiceUnavailable("Unable to generate room code, please try again")
}

func DeleteRoomCode(code string, ctx context.Context) error {
	key := "room:" + code
	return infra.Redis.Del(ctx, key).Err()
}

func RoomExists(code string, ctx context.Context) (bool, error) {
	key := "room:" + code
	exists, err := infra.Redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func randomCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = codeChars[rand.Intn(len(codeChars))]
	}
	return string(b)
}
