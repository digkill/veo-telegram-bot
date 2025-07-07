FROM golang:1.21
WORKDIR /app
COPY . .
RUN go build -o veo-bot ./cmd
# Добавляем .env
COPY .env .env
CMD ["./veo-bot"]
