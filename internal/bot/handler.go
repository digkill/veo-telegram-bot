package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/generator"
	storage "github.com/digkill/veo-telegram-bot/internal/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
	"strings"
	"time"
)

type LogEntry struct {
	Time      string `json:"time"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username,omitempty"`
	Action    string `json:"action"`
	Message   string `json:"message,omitempty"`
	Prompt    string `json:"prompt,omitempty"`
	Success   bool   `json:"success"`
	VideoPath string `json:"video_path,omitempty"`
	Error     string `json:"error,omitempty"`
}

func logToFile(entry LogEntry) {
	entry.Time = time.Now().Format("2006-01-02 15:04:05")
	data, _ := json.Marshal(entry)
	f, _ := os.OpenFile("storage/logs/logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.Write(append(data, '\n'))
}

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message != nil {
		handleMessage(bot, update.Message)
	}

	if update.CallbackQuery != nil {
		handleCallback(bot, update.CallbackQuery)
	}

	if update.PreCheckoutQuery != nil {
		resp := tgbotapi.PreCheckoutConfig{
			PreCheckoutQueryID: update.PreCheckoutQuery.ID,
			OK:                 true,
		}
		bot.Send(resp)
	}

	if update.Message != nil && update.Message.SuccessfulPayment != nil {
		handlePayment(bot, update.Message)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text
	userID := msg.From.ID
	username := msg.From.UserName

	logToFile(LogEntry{UserID: userID, Username: username, Action: "user_message", Message: text, Success: true})

	if text == "/start" {
		bot.Send(tgbotapi.NewMessage(chatID, "Привет! Напиши промт для генерации видео, например:\n\n`Кот на пляже на закате #9:16`\n\nили используй команду /buy чтобы купить кредиты 💳"))
		return
	}

	if text == "/buy" {
		showBuyOptions(bot, chatID)
		return
	}

	go func() {
		if err := storage.EnsureUser(userID, username); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка: "+err.Error()))
			logToFile(LogEntry{UserID: userID, Username: username, Action: "ensure_user", Prompt: text, Success: false, Error: err.Error()})
			return
		}

		bot.Send(tgbotapi.NewMessage(chatID, "🎬 Генерирую видео, подожди 30–60 секунд..."))

		videoPath, err := generator.GenerateVideo(text, userID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Видео не удалось сгенерировать: "+err.Error()))
			logToFile(LogEntry{UserID: userID, Username: username, Action: "generate", Prompt: text, Success: false, Error: err.Error()})
			if videoPath != "" {
				_ = os.Remove(videoPath)
			}
			return
		}

		if err := storage.SubtractCredits(userID, 150); err != nil {
			if errors.Is(err, storage.ErrInsufficientCredits) {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Видео сгенерировано, но у тебя недостаточно кредитов для его получения. Купи пакет через /buy"))
				logToFile(LogEntry{UserID: userID, Username: username, Action: "insufficient_credits", Prompt: text, Success: false, VideoPath: videoPath})
				_ = os.Remove(videoPath)
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при списании: "+err.Error()))
				logToFile(LogEntry{UserID: userID, Username: username, Action: "debit_failed", Prompt: text, Success: false, Error: err.Error()})
			}
			return
		}

		video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(videoPath))
		video.Caption = "Вот твоё видео!"
		bot.Send(video)

		logToFile(LogEntry{UserID: userID, Username: username, Action: "generate_success", Prompt: text, Success: true, VideoPath: videoPath})
	}()
}

func showBuyOptions(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Выбери пакет кредитов 💳")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("200 кр. — 450 ₽", "buy_200"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("500 кр. — 900 ₽", "buy_500"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1200 кр. — 1800 ₽", "buy_1200"),
		),
	)
	bot.Send(msg)
}

func handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery) {
	var credits, price int
	var label, startParam string

	switch cb.Data {
	case "buy_200":
		credits, price, label, startParam = 200, 45000, "200 кредитов", "buy_200"
	case "buy_500":
		credits, price, label, startParam = 500, 90000, "500 кредитов", "buy_500"
	case "buy_1200":
		credits, price, label, startParam = 1200, 180000, "1200 кредитов", "buy_1200"
	default:
		return
	}

	invoice := tgbotapi.InvoiceConfig{
		BaseChat:            tgbotapi.BaseChat{ChatID: cb.Message.Chat.ID},
		Title:               "Покупка кредитов",
		Description:         fmt.Sprintf("Пакет: %s", label),
		Payload:             fmt.Sprintf("credits_%d", credits),
		ProviderToken:       os.Getenv("PROVIDER_TOKEN"),
		StartParameter:      startParam,
		Currency:            "RUB",
		Prices:              []tgbotapi.LabeledPrice{{Label: label, Amount: price}},
		SuggestedTipAmounts: []int{},
		IsFlexible:          false,
	}

	if _, err := bot.Send(invoice); err != nil {
		bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "❌ Ошибка при отправке инвойса: "+err.Error()))
		logToFile(LogEntry{UserID: cb.From.ID, Username: cb.From.UserName, Action: "invoice_error", Success: false, Error: err.Error()})
	} else {
		logToFile(LogEntry{UserID: cb.From.ID, Username: cb.From.UserName, Action: "invoice_sent", Message: label, Success: true})
	}
}

func handlePayment(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	payload := msg.SuccessfulPayment.InvoicePayload
	userID := msg.From.ID
	username := msg.From.UserName

	if strings.HasPrefix(payload, "credits_") {
		parts := strings.Split(payload, "_")
		credits, _ := strconv.Atoi(parts[1])

		if err := storage.AddCredits(userID, username, credits); err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Ошибка при начислении кредитов"))
			logToFile(LogEntry{UserID: userID, Username: username, Action: "payment_failed", Success: false, Error: err.Error()})
			return
		}

		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ %d кредитов зачислено!", credits)))
		logToFile(LogEntry{UserID: userID, Username: username, Action: "payment_success", Message: fmt.Sprintf("%d credits", credits), Success: true})
	}
}
