package generator

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/digkill/veo-telegram-bot/internal/db"
	"github.com/digkill/veo-telegram-bot/internal/utils"
	"log"
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

func GenerateVideo(prompt string, telegramID int64) (string, error) {
	aspectRatio, cleanPrompt := extractAspectRatio(prompt)

	tplBytes, err := os.ReadFile("templates/request.tpl.json")
	if err != nil {
		return "", err
	}
	tpl, err := template.New("request").Parse(string(tplBytes))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, map[string]string{
		"Prompt":      strings.TrimSpace(cleanPrompt),
		"AspectRatio": aspectRatio,
	})
	if err != nil {
		return "", err
	}

	err = os.WriteFile("request.json", buf.Bytes(), 0644)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("bash", "-c", fmt.Sprintf(`
		curl -s -X POST \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(gcloud auth print-access-token)" \
		"https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:predictLongRunning" \
		-d @request.json`, apiEndpoint, projectID, locationID, modelID))

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ошибка curl: %w", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", err
	}

	opID, ok := resp["name"].(string)
	if !ok {
		return "", fmt.Errorf("не удалось извлечь operation ID")
	}

	// polling
	for i := 0; i < 12; i++ {
		time.Sleep(10 * time.Second)
		fetchOut, err := fetchOperation(opID)
		if err != nil {
			return "", err
		}

		var fetchResp map[string]interface{}
		if err := json.Unmarshal(fetchOut, &fetchResp); err != nil {
			return "", err
		}

		predictions, ok := fetchResp["predictions"].([]interface{})
		if ok && len(predictions) > 0 {
			firstPred := predictions[0].(map[string]interface{})
			videoBase64, ok := firstPred["videoBytes"].(string)
			if !ok {
				return "", fmt.Errorf("videoBytes отсутствует")
			}

			videoData, err := base64.StdEncoding.DecodeString(videoBase64)
			if err != nil {
				return "", err
			}

			// создаём директорию
			dir := fmt.Sprintf("storage/media/%d", telegramID)
			os.MkdirAll(dir, 0755)

			filename := fmt.Sprintf("%s/video_%d.mp4", dir, time.Now().Unix())
			if err := os.WriteFile(filename, videoData, 0644); err != nil {
				return "", err
			}

			// логируем генерацию
			_, _ = db.DB.Exec(`
				INSERT INTO user_logs (user_id, action_type, prompt, success, video_path)
				VALUES (?, 'generation', ?, 1, ?)`,
				telegramID, prompt, filename,
			)

			return filename, nil
		}
	}

	// лог ошибки
	_, _ = db.DB.Exec(`
		INSERT INTO user_logs (user_id, action_type, prompt, success)
		VALUES (?, 'generation', ?, 0)`,
		telegramID, prompt,
	)

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
		log.Fatalf("❌ Не удалось получить access token: %v", err)
	}
	return strings.TrimSpace(string(out))
}
