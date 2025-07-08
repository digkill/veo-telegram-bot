package bot

import (
	"errors"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/generator"
	"github.com/digkill/veo-telegram-bot/internal/logger"
	"github.com/digkill/veo-telegram-bot/internal/repository"
	"github.com/digkill/veo-telegram-bot/internal/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
	"strings"
)

const welcomeMessage = `👋 Привет! Я Veo Telegram Bot — твой AI-помощник по генерации видео.

🎥 Просто отправь мне текст (промт), и я создам видео с помощью Google Veo.

📏 Укажи формат:
• Пример: *Кот на пляже на закате #9:16*
• Поддержка: #9:16, #16:9, #1:1

💳 Напиши /buy, чтобы пополнить кредиты.
📖 Напиши /help, чтобы узнать все команды.
`

const helpMessage = `📖 Список команд:

/start — приветственное сообщение  
/help — показать это меню  
/balance — твой текущий баланс  
/buy — купить кредиты  
/ping — проверить статус бота

💬 Просто отправь промт, например:
*Фэнтези лес в лунном свете #16:9*

🎞️ Через минуту ты получишь AI-видео!
`

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	logger.LogUpdate(update)

	if update.Message != nil {
		handleMessage(bot, update.Message)
	}

	if update.CallbackQuery != nil {
		logger.LogCallback(update.CallbackQuery)
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
	logger.LogMessage(msg)

	if msg.SuccessfulPayment != nil {
		return // не запускаем генерацию на сообщении об оплате
	}

	chatID := msg.Chat.ID
	text := msg.Text
	userID := msg.From.ID
	username := msg.From.UserName

	switch text {
	case "/start":
		msg := tgbotapi.NewMessage(chatID, welcomeMessage)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return

	case "/help":
		msg := tgbotapi.NewMessage(chatID, helpMessage)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		return

	case "/buy":
		showBuyOptions(bot, chatID)
		return

	case "/balance":
		balance, err := repository.GetBalance(userID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при получении баланса"))
			return
		}
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("💰 У тебя %d кредитов.", balance)))
		return
	}

	go func() {
		if err := repository.EnsureUser(userID, username); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка: "+err.Error()))
			return
		}

		balance, err := repository.GetBalance(userID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Не удалось получить баланс"))
			return
		}

		if balance < 150 {
			bot.Send(tgbotapi.NewMessage(chatID, "😢 У тебя недостаточно кредитов для генерации видео (нужно 150). Пополни баланс через /buy"))
			return
		}

		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🎬 Генерирую видео (150 кр.)… У тебя %d кр. осталось.", balance)))

		imageBase64 := ""
		if msg.Photo != nil && len(msg.Photo) > 0 {
			photo := msg.Photo[len(msg.Photo)-1]
			file, err := bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
			if err == nil {
				url := file.Link(bot.Token)
				imageBase64, err = utils.DownloadAndEncodeImage(url)
				if err != nil {
					logger.LogError("image", map[string]interface{}{
						"user_id": userID,
						"error":   err.Error(),
					})
				}
			}
		}

		videoPath, err := generator.GenerateVideo(text, userID, imageBase64)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Видео не удалось сгенерировать: "+err.Error()+"\n\n💡 Не волнуйся, кредиты не списаны. Попробуй переформулировать запрос или используй другой промт."))
			repository.LogAction(userID, "generation_failed", text, false, "")
			if videoPath != "" {
				_ = os.Remove(videoPath)
			}
			return
		}

		if err := repository.SubtractCredits(userID, 150); err != nil {
			if errors.Is(err, repository.ErrInsufficientCredits) {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Видео сгенерировано, но у тебя недостаточно кредитов для его получения. Купи пакет через /buy"))
				repository.LogAction(userID, "delivery_failed", text, false, videoPath)
				_ = os.Remove(videoPath)
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при списании: "+err.Error()))
			}
			return
		}

		video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(videoPath))
		video.Caption = "Вот твоё видео!"
		bot.Send(video)

		newBalance, _ := repository.GetBalance(userID)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Успешно! Остаток: %d кр.", newBalance)))
	}()
}

func showBuyOptions(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Выбери пакет кредитов 💳")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("200 кр. — 450 ₽", "buy_200")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("500 кр. — 900 ₽", "buy_500")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("1200 кр. — 1800 ₽", "buy_1200")),
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
	}
}

func handlePayment(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	logger.LogPayment(msg)

	payload := msg.SuccessfulPayment.InvoicePayload
	userID := msg.From.ID
	username := msg.From.UserName

	if strings.HasPrefix(payload, "credits_") {
		parts := strings.Split(payload, "_")
		credits, _ := strconv.Atoi(parts[1])

		if err := repository.AddCredits(userID, username, credits); err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Ошибка при начислении кредитов"))
			return
		}

		// Получим обновлённый баланс
		balance, err := repository.GetBalance(userID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ %d кредитов зачислено!\n⚠️ Но не удалось получить текущий баланс.", credits)))
			return
		}

		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ %d кредитов зачислено!\n💰 Текущий баланс: %d кр.", credits, balance)))
	}
}
