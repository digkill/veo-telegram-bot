package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"os"
	"time"
)

var rdb *redis.Client
var ctx = context.Background()

func Init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),     // пример: "localhost:6379"
		Password: os.Getenv("REDIS_PASSWORD"), // можно оставить пустым
		DB:       0,
	})
}

// promptData — структура хранения данных генерации
type promptData struct {
	Prompt      string `json:"prompt"`
	ImageBase64 string `json:"image_base64,omitempty"`
}

// StorePromptRequest сохраняет текст и изображение во временное хранилище (TTL 10 минут)
func StorePromptRequest(userID int64, prompt, imageBase64 string) error {
	data := promptData{
		Prompt:      prompt,
		ImageBase64: imageBase64,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	key := fmt.Sprintf("prompt:%d", userID)
	return rdb.Set(ctx, key, jsonData, 10*time.Minute).Err()
}

// GetPromptData возвращает сохранённые данные по userID
func GetPromptData(userID int64) (string, string, error) {
	key := fmt.Sprintf("prompt:%d", userID)
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", "", err
	}

	var data promptData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return "", "", fmt.Errorf("unmarshal error: %w", err)
	}

	return data.Prompt, data.ImageBase64, nil
}

// ClearPrompt удаляет сохранённый промт
func ClearPrompt(userID int64) {
	key := fmt.Sprintf("prompt:%d", userID)
	_ = rdb.Del(ctx, key).Err()
}
