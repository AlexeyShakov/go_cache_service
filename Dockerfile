# Dockerfile собирает и упаковывает Go Cache Service
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

# Путь до main-пакета (можно переопределить через build arg)
ARG MAIN=./cmd/app

# Сборка бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/app ${MAIN}

# ---------- stage 2: runtime ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app

# Копируем только готовый бинарник
COPY --from=builder /out/app /app/app

# Порт HTTP-сервера
EXPOSE 8080

# Точка входа
ENTRYPOINT ["/app/app"]