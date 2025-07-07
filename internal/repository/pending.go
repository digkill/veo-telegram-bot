package repository

import (
	"github.com/digkill/veo-telegram-bot/internal/db"
)

func SavePendingVideo(userID int64, prompt, videoPath string) error {
	_, err := db.DB.Exec(`
		INSERT INTO pending_videos (user_id, prompt, video_path)
		VALUES (?, ?, ?)`,
		userID, prompt, videoPath,
	)
	return err
}

func GetPendingVideos(userID int64) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT video_path FROM pending_videos WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		videos = append(videos, path)
	}
	return videos, nil
}

func ClearPendingVideos(userID int64) {
	db.DB.Exec(`DELETE FROM pending_videos WHERE user_id = ?`, userID)
}
