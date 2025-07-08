DB_URL = $(shell grep DB_DSN .env | cut -d '=' -f2)

migrate-up:
	goose -dir ./internal/db/migrations mysql "$(DB_URL)" up

migrate-down:
	goose -dir ./internal/db/migrations mysql "$(DB_URL)" down

install-deps:
	sudo apt update && sudo apt install -y ffmpeg supervisor unzip curl jq

install-gcloud:
	curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-471.0.0-linux-x86_64.tar.gz
	tar -xf google-cloud-cli-*.tar.gz
	./google-cloud-sdk/install.sh
	echo 'source ~/google-cloud-sdk/path.bash.inc' >> ~/.bashrc
	source ~/.bashrc

gcloud-auth:
	gcloud init
	gcloud auth application-default login
