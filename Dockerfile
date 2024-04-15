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
WORKDIR /root/
COPY --from=builder /app/server .
CMD ["./server"]

