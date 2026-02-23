# Сервис кэширования

<details>
<summary><strong>ТЗ</strong></summary>

## Требования
Необходимо реализовать сервис кэширования на языке Go.

Сервис должен использовать двухуровневую модель хранения данных:

* основной источник данных — Redis;

* локальный in-memory кэш на основе map.

Логика работы кэша:

* При запросе значения сначала выполняется поиск в локальном кэше.

* При отсутствии значения производится обращение к Redis.

* Полученное значение сохраняется в локальном кэше.

Одновременно с основным процессом должен запускаться фоновый воркер, который с заданной периодичностью обращается к Redis и обновляет значения, уже присутствующие в локальном кэше.

Локальный кэш должен быть потокобезопасным
</details>

<details>
<summary><strong>Реализация</strong></summary>

## Архитектура проекта
```
              ┌────────────────────┐
              │     Refresher      │
              │  (background job   |
              │ interval + jitter) │
              └─────────┬──────────┘
                        │
                        ▼
Client → HTTP (Echo) → CacheService ─────────→ RedisRepository → Redis
                        │
                        ▼
                   InMemoryCache
```

## Структура проекта
Проект организован по слоистой архитектуре: доменная логика отделена от инфраструктуры (Redis/HTTP), а точки входа (cmd/*) 
содержат только сборку зависимостей и запуск компонентов.

### Точки входа
* cmd/app/ — основной исполняемый файл сервиса. main.go — собирает приложение: загружает конфиги, создаёт Redis-клиент/репозиторий, 
    in-memory кэш и CacheService, запускает HTTP-сервер и фонового воркера, настраивает graceful shutdown через context.

* cmd/seed/ — CLI для заполнения Redis тестовыми данными. main.go — читает dataset из JSON-файла (путь задаётся SEED_DATASET_PATH), 
   подключается к Redis и записывает значения в Redis Hash. Удобно для ручного тестирования через Postman и для интеграционных тестов.

### Доменный слой
**internal/domain/** — доменные типы и ошибки, не зависящие от инфраструктуры.

* errors.go — набор доменных ошибок/категорий (ErrNotFound, ErrUnavailable, ErrTimeout, ErrCancelled, ErrPermanent, ErrRepeatedRequest и т.д.). 
    Эти ошибки используются во всех слоях как единый “язык” ошибок.

* types.go — базовые доменные типы (например Key, Value) и вспомогательные определения, чтобы не “тащить” инфраструктурные 
    типы в бизнес-логику.

#### Сервисный слой
**internal/domain/service/** — реализация механизма кэширования и фонового обновления.

* cache_service.go — основной сервис CacheService:

    * read-through логика: сначала in-memory, затем Redis;

    * fail-fast дедупликация параллельных запросов на один ключ через inFlight;

    * батч-обновление ключей в Refresh.

* refresher.go — фоновый воркер:

    * периодический вызов Refresh по таймеру;

    * джиттер, чтобы разные инстансы не обновляли кэш синхронно;

    * context.WithTimeout на каждую итерацию обновления;

    * корректное завершение при отмене контекста.

* config.go — конфигурация воркера (интервал, таймаут, джиттер и т.д.) и загрузка из env.

### In-memory кэш
**internal/cache/** — реализация локального кэша в памяти.

* cache.go — InMemoryCache на базе map + sync.RWMutex:

    * быстрые чтения (RLock);

    * безопасные записи (Lock);

    * операции массового обновления (замена значений по ключам).

### Инфраструктурный слой
**internal/infrastructure/http/** — HTTP-обвязка вокруг сервиса:
    
* handlers.go — хендлеры Echo:

  * парсинг параметров;
  * вызов CacheService;
  * маппинг доменных ошибок в HTTP-коды (например ErrNotFound → 404).

* server.go — инициализация Echo-сервера, middleware, маршруты.

**internal/infrastructure/redis/** — работа с Redis и маппинг ошибок.

* client.go — создание Redis-клиента, проверка доступности через Ping.

* repository.go — RedisRepository (доступ к данным через Redis Hash):

    * GetByKey / GetByKeys;

    * преобразование ошибок Redis в доменные ошибки.

* errors.go — логика классификации/нормализации Redis-ошибок (auth/permanent, unavailable, internal и т.д.).

* config.go — конфиг Redis (адрес, пароль, db, hash key, batch size) и загрузка из env.

### Заполнение БД данными
**internal/seed/** — библиотека для записи dataset в Redis.

* seed.go — функция Run(...), которая принимает уже распарсенные данные и записывает их в Redis Hash. 
    Используется CLI из cmd/seed.

### Тестовые данные
**internal/testdata/** — набор тестовых данных для сидирования.

* seed.json — dataset для заполнения Redis (используется cmd/seed). Подходит как для ручной проверки через Postman, так и для интеграционных тестов.

## Стек технологий
| Компонент            | Используемое решение                     |
|----------------------|------------------------------------------|
| Язык                 | Go 1.25                                  |
| Web Framework        | Echo v5                                  |
| Redis Client         | go-redis/v9                              |
| Конфигурация         | cleanenv                                 |
| Логирование          | slog (стандартная библиотека)            |
| Контейнеризация      | Docker + docker-compose                  |
| Тестовые данные      | JSON dataset (seed.json)                 |

## Запуск приложения
Проект полностью контейнеризирован: Redis, backend и seed-утилита запускаются в Docker и управляются через docker-compose.

### Подготовка окружения
Создайте файл .env в корне проекта и заполните переменные (пример ниже).
docker-compose.yml автоматически подхватывает .env через env_file.

```
REDIS_ADDR=redis:6379
REDIS_USER=
REDIS_PASSWORD=password123
REDIS_DB=0
REDIS_HASH_KEY=cache
REDIS_KEYS_BATCH=100

REFRESH_TIMEOUT=200ms
REFRESH_INTERVAL=30s
JITTER_MAX_VALUE=3

LOG_LEVEL=debug
```
* REDIS_ADDR — адрес Redis. В docker-compose используется имя сервиса redis, поэтому значение redis:6379.

* REDIS_USER — имя пользователя Redis (опционально). Если Redis без ACL-пользователей, оставляется пустым.

* REDIS_PASSWORD — пароль для подключения к Redis. Должен совпадать с паролем, который передаётся Redis контейнеру.

* REDIS_DB — номер базы Redis (по умолчанию 0).

* REDIS_HASH_KEY — имя Redis Hash, в котором хранятся значения (ключи API лежат внутри этого hash).

* REDIS_KEYS_BATCH — размер батча при обновлении кэша воркером (сколько ключей за один запрос HMGET).

* REFRESH_TIMEOUT — таймаут одной итерации обновления кэша (используется context.WithTimeout).

* REFRESH_INTERVAL — интервал между обновлениями кэша.

* JITTER_MAX_VALUE — максимальный джиттер в минутах, добавляемый к интервалу обновления, чтобы разные инстансы не обновляли кэш синхронно.

* LOG_LEVEL — уровень логирования (debug, info, warn, error). Для разработки удобно debug, для production обычно info

### Запуск
1. Сборка образов
```
docker compose build
```
2. Запуск основного приложения
```
docker compose up redis backend
```
Для запуска приложения в фоновом режиме:
```
docker compose up -d redis backend
```
3. Заполнение Redis тестовыми данными
```
docker compose run --rm seed
```
</details>