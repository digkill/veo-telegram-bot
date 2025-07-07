package repository

import (
	_ "database/sql"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/db"
)

func EnsureUser(telegramID int64, username string) error {
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id = ?)", telegramID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("ошибка при проверке пользователя: %v", err)
	}

	if !exists {
		_, err := db.DB.Exec(`INSERT INTO users (telegram_id, username) VALUES (?, ?)`, telegramID, username)
		if err != nil {
			return fmt.Errorf("ошибка при создании пользователя: %v", err)
		}
	}
	return nil
}
