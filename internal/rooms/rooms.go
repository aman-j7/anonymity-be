package rooms

import (
	"anonymity/constants"
	"anonymity/internal/infra"
	"context"
	"log"
	"math/rand"
	"strconv"
	"time"
)

const (
	codeChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength = 6
	maxRetries = 5
	roomTTL    = 1 * time.Hour
)

func Init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	updateRoomCodeBatchSize(constants.RoomBatchSize)
	addRoomCodesInBucket(constants.RoomBatchSize)
}

func RoomExists(code string, ctx context.Context) (bool, error) {
	key := "room:" + code
	exists, err := infra.Redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func GetRoomCode(ctx context.Context) (string, error) {
	val, err := infra.Redis.SPop(ctx, constants.RoomCodeBucketKey).Result()
	go checkAvailableRoomCode()
	return val, err
}

func GenerateBatchRoomCode(size int64) []interface{} {
	codes := make([]string, 0, size)
	unique := make(map[string]struct{})

	for int64(len(codes)) < size {
		code := randomCode()

		if _, exists := unique[code]; exists {
			continue
		}

		unique[code] = struct{}{}
		codes = append(codes, code)
	}

	return toInterfaceSlice(codes)
}

func randomCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = codeChars[rand.Intn(len(codeChars))]
	}
	return string(b)
}

func toInterfaceSlice(arr []string) []interface{} {
	res := make([]interface{}, len(arr))
	for i, v := range arr {
		res[i] = v
	}
	return res
}

func checkAvailableRoomCode() {
	context := context.Background()
	roomCodeCount, roomCodesError := infra.Redis.SCard(context, constants.RoomCodeBucketKey).Result()
	if roomCodesError != nil {
		log.Printf("Error occurred fetching room count: %v", roomCodesError)
		return
	}

	currentBatchSize, batchSizeError := infra.Redis.Get(context, constants.RoomBucketSizeKey).Result()
	if batchSizeError != nil {
		log.Printf("Error occurred fetching room batch size: %v", batchSizeError)
		return
	}
	size, _ := strconv.ParseInt(currentBatchSize, 10, 64)

	if size >= constants.MaxRoomCount {
		log.Printf("Max room limit %d reached", constants.MaxRoomCount)
		return
	}

	if (roomCodeCount/size)*100 >= 80 {
		availableSize := constants.MaxRoomCount - size
		if availableSize < constants.RoomBatchSize {
			currentSize := size + availableSize
			updateRoomCodeBatchSize(currentSize)
			addRoomCodesInBucket(currentSize)
		} else {
			currentSize := availableSize - constants.RoomBatchSize + size
			updateRoomCodeBatchSize(currentSize)
			addRoomCodesInBucket(constants.RoomBatchSize)
		}
	}
}

func updateRoomCodeBatchSize(size int64) {
	_, err := infra.Redis.Set(context.Background(), constants.RoomBucketSizeKey, size, 0).Result()
	if err != nil {
		log.Printf("Error occurred on room bucket size initialization: %v", err)
		return
	}
}

func addRoomCodesInBucket(size int64) {
	codes := GenerateBatchRoomCode(size)
	err := infra.Redis.SAdd(context.Background(), constants.RoomCodeBucketKey, codes...).Err()

	if err != nil {
		log.Printf("Room code token bucket initialization failed %v", err)
		return
	}
	log.Printf("%d new room codes added to bucket", size)
}
