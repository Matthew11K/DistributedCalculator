# Используем официальный образ Go
FROM golang:1.22.2 as builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем исходный код
COPY . .

# Скачиваем зависимости
RUN go mod tidy

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# Финальный этап сборки
FROM alpine:latest
RUN apk add --no-cache curl \
    && curl -L https://github.com/jwilder/dockerize/releases/download/v0.6.1/dockerize-alpine-linux-amd64-v0.6.1.tar.gz | tar xz -C /usr/local/bin

WORKDIR /root/
COPY --from=builder /app/server .
CMD ["dockerize", "-wait", "tcp://postgres:5432", "-wait", "tcp://rabbitmq:5672", "-timeout", "60s", "./server"]