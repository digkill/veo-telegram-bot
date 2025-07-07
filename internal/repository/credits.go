package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/db"
)

var ErrInsufficientCredits = errors.New("недостаточно кредитов")

// GetCredits — получить баланс по telegram_id
func GetCredits(telegramID int64) (int, error) {
	var credits int
	err := db.DB.QueryRow("SELECT credits FROM users WHERE telegram_id = ?", telegramID).Scan(&credits)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return credits, err
}

// AddCredits — начислить кредиты, создать пользователя если нужно
func AddCredits(telegramID int64, username string, amount int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO users (telegram_id, username, credits)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE credits = credits + ?, username = VALUES(username)`,
		telegramID, username, amount, amount,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SubtractCredits — безопасно списать кредиты, если хватает
func SubtractCredits(telegramID int64, amount int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current int
	err = tx.QueryRow("SELECT credits FROM users WHERE telegram_id = ? FOR UPDATE", telegramID).Scan(&current)
	if err != nil {
		return err
	}

	if current < amount {
		return ErrInsufficientCredits
	}

	_, err = tx.Exec("UPDATE users SET credits = credits - ? WHERE telegram_id = ?", amount, telegramID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func CheckCredits(telegramID int64) (int, error) {
	var credits int

	err := db.DB.QueryRow("SELECT credits FROM users WHERE telegram_id = ?", telegramID).Scan(&credits)
	if err == sql.ErrNoRows {
		// ❗ если пользователь не найден — регистрируем
		_, err := db.DB.Exec("INSERT INTO users (telegram_id) VALUES (?)", telegramID)
		if err != nil {
			return 0, fmt.Errorf("❌ Ошибка при автосоздании пользователя: %v", err)
		}
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("⚠️ Ошибка при проверке баланса: %v", err)
	}

	return credits, nil
}
