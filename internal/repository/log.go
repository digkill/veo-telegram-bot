package repository

import (
	"github.com/digkill/veo-telegram-bot/internal/db"
)

func LogAction(userID int64, actionType string, prompt string, success bool, videoPath string) {
	_, _ = db.DB.Exec(`
		INSERT INTO user_logs (user_id, action_type, prompt, success, video_path)
		VALUES (?, ?, ?, ?, ?)`,
		userID, actionType, prompt, success, videoPath,
	)
}
