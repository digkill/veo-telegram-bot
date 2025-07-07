package logger

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

var fileLogger *log.Logger

func Init() {
	logsDir := "./storage/logs"
	os.MkdirAll(logsDir, os.ModePerm)

	logFilePath := filepath.Join(logsDir, "logs.txt")
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Ошибка при создании файла логов: %v", err)
	}

	fileLogger = log.New(file, "", log.LstdFlags)
}

func Logf(format string, v ...interface{}) {
	fileLogger.Printf(format, v...)
}

func Log(message string) {
	fileLogger.Println(time.Now().Format("2006-01-02 15:04:05") + " " + message)
}
