package logger

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	_ "path/filepath"
	"strings"
	"time"
)

var (
	defaultLogPath  = "storage/logs/logs.txt"
	errorsLogPath   = "storage/logs/errors.log"
	paymentsLogPath = "storage/logs/payments.log"
	logLevel        = "info"
	logToStdout     = false
	generalLogger   *log.Logger
)

func Init() {
	_ = godotenv.Load() // загрузка .env

	_ = os.MkdirAll("storage/logs", os.ModePerm)

	logLevel = strings.ToLower(os.Getenv("LOG_LEVEL"))
	logToStdout = os.Getenv("LOG_STDOUT") == "true"

	logFile, err := os.OpenFile(defaultLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Ошибка открытия основного лог-файла: %v", err)
	}

	generalLogger = log.New(logFile, "", log.LstdFlags)
}

func shouldLog(level string) bool {
	order := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}
	return order[strings.ToLower(level)] >= order[logLevel]
}

func logRaw(level string, message string) {
	if !shouldLog(level) {
		return
	}
	formatted := fmt.Sprintf("[%s] %s", strings.ToUpper(level), message)
	generalLogger.Println(formatted)
	if logToStdout {
		fmt.Println(formatted)
	}
}

func Logf(format string, v ...interface{}) {
	logRaw("info", fmt.Sprintf(format, v...))
}

func Log(message string) {
	logRaw("info", message)
}

func writeJSONLog(entry map[string]interface{}, targetPath string) {
	entry["timestamp"] = time.Now().Format(time.RFC3339)
	data, err := json.Marshal(entry)
	if err != nil {
		logRaw("error", fmt.Sprintf("❌ Ошибка сериализации JSON-лога: %v", err))
		return
	}

	f, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logRaw("error", fmt.Sprintf("❌ Ошибка открытия файла лога: %v", err))
		return
	}
	defer f.Close()

	_, _ = f.Write(append(data, '\n'))

	if logToStdout {
		fmt.Println(string(data))
	}
}

func LogUpdate(update tgbotapi.Update) {
	if !shouldLog("debug") {
		return
	}
	entry := map[string]interface{}{
		"type":   "update",
		"update": update,
	}
	writeJSONLog(entry, defaultLogPath)
}

func LogMessage(msg *tgbotapi.Message) {
	entry := map[string]interface{}{
		"type":    "message",
		"user_id": msg.From.ID,
		"chat_id": msg.Chat.ID,
		"text":    msg.Text,
	}
	writeJSONLog(entry, defaultLogPath)
}

func LogCallback(cb *tgbotapi.CallbackQuery) {
	entry := map[string]interface{}{
		"type":     "callback",
		"user_id":  cb.From.ID,
		"chat_id":  cb.Message.Chat.ID,
		"data":     cb.Data,
		"username": cb.From.UserName,
	}
	writeJSONLog(entry, defaultLogPath)
}

func LogPayment(msg *tgbotapi.Message) {
	entry := map[string]interface{}{
		"type":     "payment",
		"user_id":  msg.From.ID,
		"username": msg.From.UserName,
		"amount":   msg.SuccessfulPayment.TotalAmount,
		"payload":  msg.SuccessfulPayment.InvoicePayload,
	}
	writeJSONLog(entry, paymentsLogPath)
}

func LogResponse(resp interface{}) {
	entry := map[string]interface{}{
		"type":     "response",
		"response": resp,
	}
	writeJSONLog(entry, defaultLogPath)
}

func LogError(errMsg string, ctx map[string]interface{}) {
	entry := map[string]interface{}{
		"type":    "error",
		"message": errMsg,
	}
	for k, v := range ctx {
		entry[k] = v
	}
	writeJSONLog(entry, errorsLogPath)
	logRaw("error", errMsg)
}
