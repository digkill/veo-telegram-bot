package main

import (
	"github.com/digkill/veo-telegram-bot/internal/cache"
	"github.com/digkill/veo-telegram-bot/internal/logger"
	"log"

	"github.com/digkill/veo-telegram-bot/internal/bot"
	"github.com/digkill/veo-telegram-bot/internal/db"
	"github.com/digkill/veo-telegram-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cache.Init()
	logger.Init()
	// подключаем БД
	db.Connect()

	// получаем токен из .env
	token := utils.MustGetEnv("TELEGRAM_BOT_TOKEN")

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// очень важно: включаем нужные типы обновлений
	u.AllowedUpdates = []string{"message", "callback_query", "pre_checkout_query"}

	updates := api.GetUpdatesChan(u)
	for update := range updates {
		go bot.HandleUpdate(api, update)
	}
}
