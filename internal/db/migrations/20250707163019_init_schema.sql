package logger

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	defaultLogPath   = "storage/logs/logs.txt"
	errorsLogPath    = "storage/logs/errors.log"
	paymentsLogPath  = "storage/logs/payments.log"
	generalLogger    *log.Logger
)

func Init() {
	_ = os.MkdirAll("storage/logs", os.ModePerm)

	logFile, err := os.OpenFile(defaultLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("❌ Ошибка открытия основного лог-файла: %v", err)
	}

	generalLogger = log.New(logFile, "", log.LstdFlags)
}

func Logf(format string, v ...interface{}) {
	generalLogger.Printf(format, v...)
}

func Log(message string) {
	generalLogger.Println(time.Now().Format("2006-01-02 15:04:05") + " " + message)
}

func writeJSONLog(entry map[string]interface{}, targetPath string) {
	entry["timestamp"] = time.Now().Format(time.RFC3339)
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Println("❌ Ошибка сериализации JSON-лога:", err)
		return
	}

	f, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("❌ Ошибка открытия файла лога:", err)
		return
	}
	defer f.Close()

	_, _ = f.Write(append(data, '\n'))
}

func LogUpdate(update tgbotapi.Update) {
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
                              }
