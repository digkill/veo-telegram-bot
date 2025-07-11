package bot

import (
	"encoding/json"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/cache"
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

🎥 Просто отправь мне текст (можешь с картинкой), и я создам видео.

📏 Укажи формат:
• Пример: *Кот на пляже на закате #9:16*
• Поддержка: #9:16, #16:9

💳 Напиши /buy, чтобы пополнить кредиты.
📖 Напиши /help, чтобы узнать все команды.
`

const helpMessage = `📖 Список команд:

/start — приветственное сообщение  
/help — показать это меню  
/balance — твой текущий баланс  
/buy — купить кредиты  
/ping — проверить статус бота

💬 Просто отправь текст (можешь с картинкой), например:
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
		return
	}

	chatID := msg.Chat.ID
	text := msg.Text
	userID := msg.From.ID
	username := msg.From.UserName

	// 👇 Вот это вставь прямо сюда:
	if msg.ReplyToMessage != nil && strings.Contains(msg.ReplyToMessage.Text, "укажи свой email") {
		email := strings.TrimSpace(msg.Text)
		if !strings.Contains(email, "@") {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Это не похоже на email, попробуй ещё раз."))
			return
		}

		err := repository.UpdateUserContact(userID, email, "")
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Не удалось сохранить email."))
			return
		}

		bot.Send(tgbotapi.NewMessage(chatID, "✅ Email сохранён! Теперь можешь выбрать пакет."))
		showBuyOptions(bot, chatID)
		return
	}

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

		if text == "" {
			text = msg.Caption
		}

		if err := cache.StorePromptRequest(userID, text, imageBase64); err != nil {
			logger.LogError("redis_store", map[string]interface{}{
				"user_id": userID,
				"error":   err.Error(),
			})
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при сохранении запроса"))
			return
		}

		confirmBtn := tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить генерацию", fmt.Sprintf("confirm_%d", userID))
		msg := tgbotapi.NewMessage(chatID, "🔄 Проверь промт и нажми кнопку, чтобы подтвердить генерацию:")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(confirmBtn))
		bot.Send(msg)
	}()
}

func handleCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery) {
	data := cb.Data

	if strings.HasPrefix(data, "confirm_") {
		userIDStr := strings.TrimPrefix(data, "confirm_")
		userID, _ := strconv.ParseInt(userIDStr, 10, 64)

		go func() {
			prompt, imageBase64, err := cache.GetPromptData(userID)
			if err != nil || prompt == "" {
				bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Не удалось получить данные запроса"))
				return
			}

			balance, err := repository.GetBalance(userID)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Не удалось получить баланс"))
				return
			}

			if balance < 150 {
				bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "😢 Недостаточно кредитов. Пополни баланс через /buy"))
				return
			}

			bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, fmt.Sprintf("🎬 Генерирую видео (150 кр.)… У тебя %d кр. на данный момент.", balance)))

			videoPath, err := generator.GenerateVideo(prompt, userID, imageBase64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "❌ Не удалось сгенерировать видео: "+err.Error()))
				repository.LogAction(userID, "generation_failed", prompt, false, "")
				cache.ClearPrompt(userID)
				return
			}

			if err := repository.SubtractCredits(userID, 150); err != nil {
				bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Ошибка при списании кредитов"))
				return
			}

			video := tgbotapi.NewVideo(cb.Message.Chat.ID, tgbotapi.FilePath(videoPath))
			video.Caption = "Вот твоё видео!"
			bot.Send(video)

			newBalance, _ := repository.GetBalance(userID)
			bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, fmt.Sprintf("✅ Успешно! Остаток: %d кр.", newBalance)))

			cache.ClearPrompt(userID)
		}()
		return
	}

	// обработка покупки
	var credits, price int
	var label, startParam string

	switch data {
	case "buy_200":
		credits, price, label, startParam = 200, 45000, "200 кредитов", "buy_200"
	case "buy_500":
		credits, price, label, startParam = 500, 90000, "500 кредитов", "buy_500"
	case "buy_1200":
		credits, price, label, startParam = 1200, 180000, "1200 кредитов", "buy_1200"
	default:
		return
	}

	user, err := repository.GetUserByID(cb.Message.Chat.ID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Не удалось получить данные пользователя."))

	}

	if user.Email == "" {
		msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "📧 Пожалуйста, укажи свой email, чтобы мы могли оформить чек.")
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		bot.Send(msg)
		return
	}

	// может быть "", это нормально
	//phone := cb.From.PhoneNumber

	// Сборка чека
	receiptItem := map[string]interface{}{
		"description": "Покупка кредитов VeoBot",
		"quantity":    1.0,
		"amount": map[string]interface{}{
			"value":    fmt.Sprintf("%.2f", float64(price)/100), // "450.00"
			"currency": "RUB",
		},
		"vat_code":        1,              // ставка НДС
		"payment_mode":    "full_payment", // полная оплата
		"payment_subject": "service",      // услуга
	}

	// Здесь нужно вставить email, phone и т.п. из базы или из заказа
	customer := map[string]interface{}{
		"email": user.Email,
		//"phone": phone,
	}

	// Собираем структуру provider_data
	providerData := map[string]interface{}{
		"receipt": map[string]interface{}{
			"items":           []interface{}{receiptItem},
			"tax_system_code": 6, // ОСН, актуально для РФ
			"customer":        customer,
		},
	}
	providerDataJSON, err := json.Marshal(providerData)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Ошибка при формировании чека"))
		return
	}

	// Инвойс
	// Формируем инвойс
	invoice := tgbotapi.InvoiceConfig{
		BaseChat:       tgbotapi.BaseChat{ChatID: cb.Message.Chat.ID},
		Title:          "Покупка кредитов",
		Description:    fmt.Sprintf("Пакет: %s", label),
		Payload:        fmt.Sprintf("credits_%d", credits),
		ProviderToken:  os.Getenv("PROVIDER_TOKEN"),
		Currency:       "RUB",
		Prices:         []tgbotapi.LabeledPrice{{Label: label, Amount: price}},
		StartParameter: startParam,
		// NeedEmail:      true,
		//NeedPhoneNumber:     true,
		ProviderData:        string(providerDataJSON),
		SuggestedTipAmounts: []int{}, // обязателен пустой массив, если чаевые не нужны
	}

	// Отправка инвойса
	if _, err := bot.Send(invoice); err != nil {
		logger.LogError("send_invoice", map[string]interface{}{
			"user_id": cb.Message.Chat.ID,
			"error":   err.Error(),
			"json":    string(providerDataJSON),
		})
		bot.Send(tgbotapi.NewMessage(cb.Message.Chat.ID, "❌ Ошибка при отправке инвойса: "+err.Error()))
	}
}

func showBuyOptions(bot *tgbotapi.BotAPI, chatID int64) {
	userID := chatID // если ты не используешь групповой чат

	hasEmail, err := repository.HasEmail(userID)
	if err != nil {
		logger.LogError("send_invoice", map[string]interface{}{
			"user_id": chatID,
			"error":   err.Error(),
			"json":    hasEmail,
		})
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Не удалось проверить email"))
	}

	if !hasEmail {
		// Спросим email
		msg := tgbotapi.NewMessage(chatID, "📧 Пожалуйста, укажи свой email для получения чека:")
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		bot.Send(msg)
		// При следующем сообщении будет reply_to_message — нужно отследить
		return
	}

	// Если email есть — показать варианты покупки
	msg := tgbotapi.NewMessage(chatID, "Выбери пакет кредитов 💳")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("200 кр. — 450 ₽", "buy_200")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("500 кр. — 900 ₽", "buy_500")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("1200 кр. — 1800 ₽", "buy_1200")),
	)
	bot.Send(msg)
}

func handlePayment(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	logger.LogPayment(msg)

	payload := msg.SuccessfulPayment.InvoicePayload
	userID := msg.From.ID
	username := msg.From.UserName

	var email, phone string
	if msg.SuccessfulPayment.OrderInfo != nil {
		if msg.SuccessfulPayment.OrderInfo.Email != "" {
			email = msg.SuccessfulPayment.OrderInfo.Email
		}
		if msg.SuccessfulPayment.OrderInfo.PhoneNumber != "" {
			phone = msg.SuccessfulPayment.OrderInfo.PhoneNumber
		}
	}

	if err := repository.UpdateUserContact(userID, email, phone); err != nil {
		logger.LogError("payment_contact_update", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}

	if strings.HasPrefix(payload, "credits_") {
		parts := strings.Split(payload, "_")
		credits, _ := strconv.Atoi(parts[1])

		if err := repository.AddCredits(userID, username, credits); err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Ошибка при начислении кредитов"))
			return
		}

		balance, err := repository.GetBalance(userID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ %d кредитов зачислено!\n⚠️ Но не удалось получить текущий баланс.", credits)))
			return
		}

		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ %d кредитов зачислено!\n💰 Текущий баланс: %d кр.", credits, balance)))
	}
}
