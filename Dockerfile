# Dockerfile собирает и упаковывает Go Cache Service (app + seed)
# в минимальный runtime-образ на базе Alpine Linux.

# ---------- stage 1: build ----------
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка основного приложения
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

# Сборка утилиты сидирования
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---------- stage 2: runtime ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app

# Копируем только готовые бинарники
COPY --from=builder /out/app /app/app
COPY --from=builder /out/seed /app/seed

# Порт HTTP-сервера
EXPOSE 8080

# По умолчанию запускаем backend.
# Для сидирования в docker-compose переопределяй command на ["/app/seed"].
ENTRYPOINT ["/app/app"]