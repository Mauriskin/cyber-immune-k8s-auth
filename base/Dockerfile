# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder

# Установка необходимых зависимостей (если нужны, для чистого бинарника не нужны)
RUN apk add --no-cache git

WORKDIR /src

# Кэшируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка статического бинарника (CGO_ENABLED=0 — без C зависимостей)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/service .

# Финальный образ — scratch (самый минимальный)
FROM scratch

# Копируем только бинарник
COPY --from=builder /app/service /service

# Non-root пользователь (рекомендуется 10001 для scratch)
USER 10001:10001

# Точка входа
ENTRYPOINT ["/service"]

# Порт (будем использовать 8080)
EXPOSE 8080
