package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var ctx = context.Background()

func Init() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),     // Пример: "localhost:6379"
		Password: os.Getenv("REDIS_PASSWORD"), // Может быть пустым
		DB:       0,
	})

	// Проверка подключения
	if err := Rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Не удалось подключиться к Redis: %v", err)
	} else {
		log.Println("✅ Успешное подключение к Redis")
	}
}

// promptData — структура хранения данных генерации
type promptData struct {
	Prompt      string `json:"prompt"`
	ImageBase64 string `json:"image_base64,omitempty"`
}

// StorePromptRequest сохраняет текст и изображение во временное хранилище (TTL 30 минут)
func StorePromptRequest(userID int64, prompt string, imageBase64 string) error {
	data := promptData{
		Prompt:      prompt,
		ImageBase64: imageBase64,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ Marshal error for user %d: %v\n", userID, err)
		return fmt.Errorf("marshal error: %w", err)
	}

	key := fmt.Sprintf("prompt:%d", userID)
	err = Rdb.Set(ctx, key, jsonData, 30*time.Minute).Err()
	if err != nil {
		log.Printf("❌ Redis SET error for user %d: %v\n", userID, err)
		return fmt.Errorf("redis set error: %w", err)
	}

	log.Printf("✅ Prompt saved for user %d: %s\n", userID, prompt)
	return nil
}

// GetPromptData возвращает сохранённые данные по userID
func GetPromptData(userID int64) (string, string, error) {
	key := fmt.Sprintf("prompt:%d", userID)
	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("❌ Redis GET error for user %d: %v\n", userID, err)
		return "", "", err
	}

	var data promptData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		log.Printf("❌ Unmarshal error for user %d: %v\n", userID, err)
		return "", "", fmt.Errorf("unmarshal error: %w", err)
	}

	log.Printf("📦 Prompt retrieved for user %d: %s\n", userID, data.Prompt)
	return data.Prompt, data.ImageBase64, nil
}

// ClearPrompt удаляет сохранённый промт
func ClearPrompt(userID int64) {
	key := fmt.Sprintf("prompt:%d", userID)
	err := Rdb.Del(ctx, key).Err()
	if err != nil {
		log.Printf("⚠️ Redis DEL error for user %d: %v\n", userID, err)
	} else {
		log.Printf("🧹 Prompt cleared for user %d\n", userID)
	}
}
