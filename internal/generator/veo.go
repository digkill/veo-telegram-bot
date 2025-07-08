package generator

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/db"
	"github.com/digkill/veo-telegram-bot/internal/logger"
	"github.com/digkill/veo-telegram-bot/internal/utils"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

var (
	projectID   = utils.MustGetEnv("PROJECT_ID")
	locationID  = utils.MustGetEnv("LOCATION_ID")
	apiEndpoint = utils.MustGetEnv("API_ENDPOINT")
	modelID     = utils.MustGetEnv("MODEL_ID")
)

func extractAspectRatio(prompt string) (string, string) {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "#9:16"):
		return "9:16", strings.ReplaceAll(prompt, "#9:16", "")
	case strings.Contains(lower, "#1:1"):
		return "1:1", strings.ReplaceAll(prompt, "#1:1", "")
	case strings.Contains(lower, "#4:5"):
		return "4:5", strings.ReplaceAll(prompt, "#4:5", "")
	case strings.Contains(lower, "#16:9"):
		return "16:9", strings.ReplaceAll(prompt, "#16:9", "")
	default:
		return "16:9", prompt
	}
}

func GenerateVideo(prompt string, telegramID int64, imageBase64 string) (string, error) {
	aspectRatio, cleanPrompt := extractAspectRatio(prompt)

	tplPath := "templates/request_without_image.tpl.json"
	if strings.TrimSpace(imageBase64) != "" {
		tplPath = "templates/request_with_image.tpl.json"
	}

	tplBytes, err := os.ReadFile(tplPath)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать шаблон %s: %w", tplPath, err)
	}
	tpl, err := template.New("request").Parse(string(tplBytes))
	if err != nil {
		return "", fmt.Errorf("ошибка парсинга шаблона: %w", err)
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, map[string]string{
		"Prompt":      strings.TrimSpace(cleanPrompt),
		"AspectRatio": aspectRatio,
		"Image64":     strings.TrimSpace(imageBase64),
	})
	if err != nil {
		return "", fmt.Errorf("ошибка шаблона: %w", err)
	}

	var jsonTest map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &jsonTest); err != nil {
		logger.LogError("generator", map[string]interface{}{
			"type":    "invalid_json",
			"user_id": telegramID,
			"error":   err.Error(),
			"raw":     buf.String(),
		})
		return "", fmt.Errorf("❌ Невалидный JSON: %s", err.Error())
	}

	// 💾 Создаём временный файл
	tmpFile := fmt.Sprintf("/tmp/request_%d.json", telegramID)
	if err := os.WriteFile(tmpFile, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("не удалось сохранить временный JSON: %w", err)
	}
	defer os.Remove(tmpFile) // удалим после использования

	logger.Logf("generator", map[string]interface{}{
		"type":    "request_payload",
		"user_id": telegramID,
		"prompt":  prompt,
		"json":    buf.String(),
	})

	cmd := exec.Command("curl", "-s", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-H", "Authorization: Bearer "+getAccessToken(),
		fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:predictLongRunning",
			apiEndpoint, projectID, locationID, modelID),
		"-d", "@"+tmpFile,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.LogError("generator", map[string]interface{}{
			"type":    "curl_error",
			"stderr":  stderr.String(),
			"user_id": telegramID,
		})
		return "", fmt.Errorf("ошибка curl: %w", err)
	}

	// продолжение анализа stdout как обычно
	out := stdout.Bytes()
	logger.Logf("generator", map[string]interface{}{
		"type":     "curl_response",
		"user_id":  telegramID,
		"response": string(out),
	})

	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", err
	}
	opID, ok := resp["name"].(string)
	if !ok {
		logger.LogError("generator", map[string]interface{}{
			"type":    "missing_operation_id",
			"raw":     string(out),
			"user_id": telegramID,
		})
		return "", fmt.Errorf("не удалось извлечь operation ID")
	}

	logger.Logf("generator", map[string]interface{}{
		"type":        "operation_id",
		"user_id":     telegramID,
		"operationID": opID,
	})

	for i := 0; i < 24; i++ {
		time.Sleep(10 * time.Second)
		fetchOut, err := fetchOperation(opID)
		if err != nil {
			return "", err
		}

		var fetchResp map[string]interface{}
		if err := json.Unmarshal(fetchOut, &fetchResp); err != nil {
			return "", err
		}

		if errData, exists := fetchResp["error"].(map[string]interface{}); exists {
			message := errData["message"].(string)
			code := int(errData["code"].(float64))
			logger.LogError("generator", map[string]interface{}{
				"type":    "generation_error",
				"code":    code,
				"message": message,
				"user_id": telegramID,
				"prompt":  prompt,
			})
			_, _ = db.DB.Exec(`
				INSERT INTO user_logs (user_id, action_type, prompt, success)
				VALUES (?, 'generation_blocked', ?, 0)`, telegramID, prompt)
			return "", fmt.Errorf("⚠️ Генерация не удалась: %s", message)
		}

		if response, ok := fetchResp["response"].(map[string]interface{}); ok {
			videosRaw, ok := response["videos"]
			if !ok {
				logger.LogError("generator", map[string]interface{}{
					"type":    "missing_videos_field",
					"payload": response,
					"user_id": telegramID,
				})
				continue // Пробуем снова на следующем шаге
			}

			videos, ok := videosRaw.([]interface{})
			if !ok || len(videos) == 0 {
				logger.LogError("generator", map[string]interface{}{
					"type":    "invalid_videos_format",
					"payload": videosRaw,
					"user_id": telegramID,
				})
				continue
			}

			video, ok := videos[0].(map[string]interface{})
			if !ok {
				logger.LogError("generator", map[string]interface{}{
					"type":    "invalid_video_entry",
					"payload": videos[0],
					"user_id": telegramID,
				})
				continue
			}

			videoBase64, ok := video["bytesBase64Encoded"].(string)
			if !ok {
				logger.LogError("generator", map[string]interface{}{
					"type":    "missing_bytesBase64Encoded",
					"payload": video,
					"user_id": telegramID,
				})
				continue
			}

			// декодируем, сохраняем файл
			videoData, err := base64.StdEncoding.DecodeString(videoBase64)
			if err != nil {
				return "", fmt.Errorf("decode error: %w", err)
			}

			dir := fmt.Sprintf("storage/media/%d", telegramID)
			_ = os.MkdirAll(dir, 0755)

			filename := fmt.Sprintf("%s/video_%d.mp4", dir, time.Now().Unix())
			if err := os.WriteFile(filename, videoData, 0644); err != nil {
				return "", err
			}

			_, _ = db.DB.Exec(`
		INSERT INTO user_logs (user_id, action_type, prompt, success, video_path)
		VALUES (?, 'generation', ?, 1, ?)`,
				telegramID, prompt, filename,
			)

			return filename, nil
		}

	}

	_, _ = db.DB.Exec(`
		INSERT INTO user_logs (user_id, action_type, prompt, success)
		VALUES (?, 'generation_timeout', ?, 0)`,
		telegramID, prompt,
	)

	logger.LogError("generator", map[string]interface{}{
		"type":    "generation_timeout",
		"prompt":  prompt,
		"user_id": telegramID,
	})
	return "", fmt.Errorf("видео не сгенерировалось за отведённое время")
}

func fetchOperation(opID string) ([]byte, error) {
	jsonBody := fmt.Sprintf(`{"operationName": "%s"}`, opID)
	cmd := exec.Command("curl", "-s", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-H", "Authorization: Bearer "+getAccessToken(),
		fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:fetchPredictOperation",
			apiEndpoint, projectID, locationID, modelID),
		"-d", "@-",
	)
	cmd.Stdin = strings.NewReader(jsonBody)
	return cmd.Output()
}

func getAccessToken() string {
	out, err := exec.Command("gcloud", "auth", "print-access-token").Output()
	if err != nil {
		panic("❌ Не удалось получить access token: " + err.Error())
	}
	return strings.TrimSpace(string(out))
}
