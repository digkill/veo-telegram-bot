package bot

import (
	"errors"
	_ "fmt"
	"github.com/digkill/veo-telegram-bot/internal/generator"
	storage "github.com/digkill/veo-telegram-bot/internal/repository"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleVideoCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	userID := msg.From.ID
	username := msg.From.UserName
	chatID := msg.Chat.ID
	prompt := msg.Text

	// Проверка и списание кредитов
	err := storage.SubtractCredits(userID, 150)
	if err != nil {
		if errors.Is(err, storage.ErrInsufficientCredits) {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Недостаточно кредитов. Купи через /buy."))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Ошибка при проверке баланса: "+err.Error()))
		}
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, "🎬 Генерирую видео, подожди немного..."))

	videoPath, err := generator.GenerateVideo(prompt, userID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка генерации: "+err.Error()))
		// (при желании: можно вернуть кредиты)
		_ = storage.AddCredits(userID, username, 150)
		return
	}

	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(videoPath))
	video.Caption = "Готово! ✨"
	bot.Send(video)

	// очистка
	defer os.Remove(videoPath)
}
