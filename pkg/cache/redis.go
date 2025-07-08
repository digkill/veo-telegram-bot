package cache

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"os"
	"strings"
	"time"
)

var ctx = context.Background()
var rdb *redis.Client

func Init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),     // например: "localhost:6379"
		Password: os.Getenv("REDIS_PASSWORD"), // "" если без пароля
		DB:       0,
	})
}

func SetPrompt(userID int64, prompt, imageBase64 string) error {
	key := fmt.Sprintf("prompt:%d", userID)
	data := prompt + "|||" + imageBase64
	return rdb.Set(ctx, key, data, 10*time.Minute).Err()
}

func GetPrompt(userID int64) (string, string, error) {
	key := fmt.Sprintf("prompt:%d", userID)
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(val, "|||", 2)
	prompt := parts[0]
	image := ""
	if len(parts) == 2 {
		image = parts[1]
	}
	return prompt, image, nil
}

func DeletePrompt(userID int64) {
	key := fmt.Sprintf("prompt:%d", userID)
	_ = rdb.Del(ctx, key).Err()
}
