package repository

import (
	"database/sql"
	_ "database/sql"
	"errors"
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

func GetBalance(userID int64) (int, error) {
	var credits int
	err := db.DB.QueryRow("SELECT credits FROM users WHERE telegram_id = ?", userID).Scan(&credits)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil // пользователь пока не существует — 0 кредитов
		}
		return 0, err
	}
	return credits, nil
}

func UpdateUserContact(userID int64, email string, phone string) error {
	_, err := db.DB.Exec(`UPDATE user SET email = ?, phone = ? WHERE telegram_id = ?`, email, phone, userID)
	return err
}
