package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env файл не найден — используем переменные окружения")
	}
}

func MustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("❌ Переменная окружения %s не задана", key)
	}
	return val
}
