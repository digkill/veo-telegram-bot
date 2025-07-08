# 🎥 Veo Telegram Bot

A powerful Telegram bot written in Go, using **Google Veo 2.0 API** to generate AI-powered videos from text prompts and optional images.  
Supports payment systems (Telegram Payments and YooKassa), credit tracking, logging, and runs as a Linux daemon via **Supervisor**.

---

## 📦 Features

- ✅ Generate videos from text prompts (with optional image)
- 🧠 Google Veo 2.0 API integration
- 💳 Buy credits using Telegram Payments and YooKassa
- 📊 Track logs of all actions and errors
- 🔐 Secure credit accounting & transactions
- 🧾 Logging to file in JSON format
- 🛠 Daemon management with Supervisor
- 🐘 MySQL storage with Goose migrations

---

## ⚙️ Installation

### 1. Clone the repository

```bash
git clone https://github.com/digkill/veo-telegram-bot.git
cd veo-telegram-bot
```

### 2. Install dependencies

```bash
make install-deps
```

Installs:
- `ffmpeg`
- `jq`
- `curl`
- `supervisor`
- `unzip`

### 3. Install Google Cloud CLI

```bash
make install-gcloud
make gcloud-auth
```

---

## 🔐 .env Configuration

Create a `.env` file like this:

```env
TELEGRAM_BOT_TOKEN=123456:ABCDEF
PROVIDER_TOKEN=your_telegram_or_yookassa_token
PROJECT_ID=vigilant-augury-xxxxxx
LOCATION_ID=us-central1
API_ENDPOINT=us-central1-aiplatform.googleapis.com
MODEL_ID=veo-2.0-generate-001

DB_DSN=digkill:YOUR_PASSWORD@tcp(YOUR_DB_HOST:PORT)/veogenbot_db?parseTime=true
```

---

## 🗃 Database Migrations

### Apply migrations:

```bash
make migrate-up
```

### Rollback:

```bash
make migrate-down
```

---

## 🤖 Run the bot manually

```bash
go run cmd/main.go
```

Or compile:

```bash
go build -o bot
./bot
```

---

## 🛡 Run as a background service (Supervisor)

### Create config: `/etc/supervisor/conf.d/veo-bot.conf`

```ini
[program:veo-bot]
directory=/opt/veo-bot
command=/opt/veo-bot/bot
autostart=true
autorestart=true
stderr_logfile=/var/log/veo-bot.err.log
stdout_logfile=/var/log/veo-bot.out.log
user=www-data
environment=HOME="/home/www-data",USER="www-data"
```

### Control Supervisor:

```bash
sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl start veo-bot
```

---

## 🧪 Test generation (no Telegram)

```bash
go run cmd/generate_test.go
```

---

## 📁 Project Structure

```
.
├── cmd/
│   └── main.go             # Bot entry point
├── internal/
│   ├── bot/                # Telegram update handlers
│   ├── generator/          # Veo API curl generator
│   ├── db/                 # Goose migrations
│   ├── repository/         # User DB helpers
│   ├── logger/             # JSON logger
│   └── utils/              # Env and misc
├── templates/
│   └── request.tpl.json    # Veo prompt templates
├── storage/
│   └── logs/               # All logs in JSON
└── Makefile
```

---

## 💡 Optional extensions

- React-based admin panel
- View generation history by user
- Subscription plans (monthly limits)
- Prompt suggestion system
- Support for voice/image inputs

---

## 🆘 Troubleshooting

If something breaks:

1. Check `.env` for missing vars
2. Inspect `storage/logs/logs.txt`
3. Run `supervisorctl status`
4. Try `gcloud auth print-access-token`
5. Make sure your service account is allowlisted in GCP

---

## 📜 License

MIT © 2025 [Mediarise](https://mediarise.ru)