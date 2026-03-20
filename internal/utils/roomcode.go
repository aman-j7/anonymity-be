package utils

import (
	"math/rand"
	"time"

	"anonymity/internal/config"
	"anonymity/internal/err"
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

func GenerateRoomCode() (string, error) {
	for i := 0; i < maxRetries; i++ {
		code := randomCode()
		key := "room:" + code

		result, err := config.RedisClient.SetArgs(config.Ctx, key, "active", redis.SetArgs{
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


func DeleteRoomCode(code string) error {
	key := "room:" + code
	return config.RedisClient.Del(config.Ctx, key).Err()
}


func RoomExists(code string) (bool, error) {
	key := "room:" + code
	exists, err := config.RedisClient.Exists(config.Ctx, key).Result()
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
