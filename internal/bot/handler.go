package bot

import (
	"errors"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/generator"
	storage "github.com/digkill/veo-telegram-bot/internal/repository"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

	if text == "/start" {
		bot.Send(tgbotapi.NewMessage(chatID, "Привет! Напиши промт для генерации видео, например:\n\n`Кот на пляже на закате #9:16`\n\nили используй команду /buy чтобы купить кредиты 💳"))
		return
	}

	if text == "/buy" {
		showBuyOptions(bot, chatID)
		return
	}

	// генерация видео
	go func() {
		userID := msg.From.ID
		username := msg.From.UserName

		// ⛑️ Убедимся, что пользователь есть в БД
		if err := storage.EnsureUser(userID, username); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка: "+err.Error()))
			return
		}

		// Проверка и списание кредитов
		err := storage.SubtractCredits(userID, 150)
		if err != nil {
			if errors.Is(err, storage.ErrInsufficientCredits) {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ У тебя недостаточно кредитов. Купи пакет с помощью /buy"))
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при проверке баланса: "+err.Error()))
			}
			return
		}

		bot.Send(tgbotapi.NewMessage(chatID, "🎬 Генерирую видео, подожди 30–60 секунд..."))

		videoPath, err := generator.GenerateVideo(text, userID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка при генерации видео: "+err.Error()))
			return
		}

		video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(videoPath))
		video.Caption = "Вот твоё видео!"
		bot.Send(video)
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
	var credits int
	var price int
	var label string

	switch cb.Data {
	case "buy_200":
		credits = 200
		price = 45000
		label = "200 кредитов"
	case "buy_500":
		credits = 500
		price = 90000
		label = "500 кредитов"
	case "buy_1200":
		credits = 1200
		price = 180000
		label = "1200 кредитов"
	default:
		bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "❌ Неизвестная команда покупки"))
		return
	}

	// отправим уведомление, что сейчас покажем оплату
	bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, fmt.Sprintf("💳 Пакет: %s. Сейчас покажу форму оплаты…", label)))

	invoice := tgbotapi.NewInvoice(
		cb.Message.Chat.ID,                 // chatID
		"Покупка кредитов",                 // title
		fmt.Sprintf("Пакет: %s", label),    // description
		fmt.Sprintf("credits_%d", credits), // payload
		os.Getenv("PROVIDER_TOKEN"),        // provider_token
		"RUB",                              // currency
		"",                                 // photo_url
		[]tgbotapi.LabeledPrice{
			{
				Label:  label,
				Amount: price,
			},
		},
	)

	_, err := bot.Send(invoice)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "❌ Ошибка при создании счёта: "+err.Error()))
	}
}

func handlePayment(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	payload := msg.SuccessfulPayment.InvoicePayload
	userID := msg.From.ID
	username := msg.From.UserName

	if strings.HasPrefix(payload, "credits_") {
		parts := strings.Split(payload, "_")
		credits, _ := strconv.Atoi(parts[1])

		// Начисляем кредиты
		err := storage.AddCredits(userID, username, credits)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Ошибка при начислении кредитов"))
			return
		}

		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ %d кредитов зачислено!", credits)))
	}
}
