DB_URL = $(shell grep DB_DSN .env | cut -d '=' -f2)

migrate-up:
	goose -dir ./internal/db/migrations mysql "$(DB_URL)" up

migrate-down:
	goose -dir ./internal/db/migrations mysql "$(DB_URL)" down

