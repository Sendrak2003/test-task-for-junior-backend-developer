# Task Service

REST API модуля трекера задач медицинской информационной системы. Реализован на Go с использованием Clean Architecture.

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Переменные окружения](#переменные-окружения)
- [Локальная разработка](#локальная-разработка)
- [Тесты](#тесты)
- [Архитектура](#архитектура)
- [База данных](#база-данных)
- [API](#api)
- [Фича: периодичность задач](#фича-периодичность-задач)
- [Принятые решения](#принятые-решения)
- [Использование LLM](#использование-llm)

---

## Быстрый старт

```bash
cp .env.example .env
docker compose up --build -d
```

Приложение доступно на `http://localhost:8080`.  
Swagger UI: `http://localhost:8080/swagger/`

Для загрузки тестовых данных:

```bash
docker compose exec postgres psql -U postgres -d taskservice -f /docker-entrypoint-initdb.d/seeds.sql
```

---

## Переменные окружения

Все переменные задаются в файле `.env` (скопировать из `.env.example`).

| Переменная        | По умолчанию                                                        | Описание                              |
|-------------------|---------------------------------------------------------------------|---------------------------------------|
| `HTTP_ADDR`       | `:8080`                                                             | Адрес HTTP-сервера                    |
| `HTTP_PORT`       | `8080`                                                              | Порт на хосте для маппинга в Docker   |
| `POSTGRES_DB`     | `taskservice`                                                       | Имя базы данных                       |
| `POSTGRES_USER`   | `postgres`                                                          | Пользователь PostgreSQL               |
| `POSTGRES_PASSWORD` | `postgres`                                                        | Пароль PostgreSQL                     |
| `POSTGRES_PORT`   | `5432`                                                              | Порт PostgreSQL на хосте              |
| `DATABASE_DSN`    | `postgres://postgres:postgres@localhost:5432/taskservice?sslmode=disable` | DSN для локальной разработки |

В Docker Compose `DATABASE_DSN` переопределяется автоматически — хост `localhost` заменяется на имя сервиса `postgres` внутри Docker-сети.

---

## Локальная разработка

Требования: Go 1.23+, Docker, Make.

```bash
# Запустить только PostgreSQL
docker compose up postgres -d

# Сгенерировать OpenAPI спеку и запустить сервис
make run

# Собрать бинарник
make build
```

### Генерация OpenAPI

Спека генерируется из аннотаций в коде с помощью [swaggo/swag](https://github.com/swaggo/swag). Это решение было предложено в процессе разработки — вместо ручного поддержания `openapi.json` аннотации в хендлерах становятся единственным источником истины. При сборке Docker-образа `swag init` запускается автоматически перед `go build`, поэтому спека всегда актуальна.

```bash
make swag
```

При сборке Docker-образа генерация запускается автоматически (`swag init` в `Dockerfile`).

---

## Тесты

```bash
# Запустить все тесты локально
go test ./internal/...

# Запустить тесты в Docker (без локального Go)
docker run --rm -v $(pwd):/src -w /src golang:1.23.0-alpine go test ./internal/...
```

### Структура тестов

Тесты написаны в виде **table-driven** структур — каждый тест содержит таблицу кейсов (`[]struct{...}`), что позволяет добавлять новые сценарии без дублирования кода. Для изоляции usecase-слоя используется in-memory mock-репозиторий.

### Покрытие

| Пакет | Тесты |
|-------|-------|
| `internal/usecase/task` | Валидация recurrence: 16 невалидных кейсов, 10 валидных; дедупликация и сортировка списков; сервис с mock-репозиторием (create/update/delete recurrence) |
| `internal/transport/http/handlers` | Маппинг domain↔DTO для всех 4 типов, List/Get с recurrence, 400 при невалидной периодичности |

---

## Архитектура

Проект следует **Clean Architecture** с однонаправленным потоком зависимостей:

```
Transport (HTTP) → Usecase → Domain
                ↓
           Repository → PostgreSQL
```

```
cmd/api/                        — точка входа, DI-сборка
internal/
  domain/task/                  — Task, Status, RecurrenceSettings, ошибки домена
  usecase/task/                 — бизнес-логика, валидация, интерфейсы портов
  repository/postgres/          — SQL-запросы, сериализация JSONB
  infrastructure/postgres/      — pgxpool, настройки пула соединений
  transport/http/
    handlers/                   — HTTP-обработчики, DTO, маппинг
    docs/                       — OpenAPI спека, Swagger UI
    router.go                   — маршруты, CORS и recovery middleware
migrations/                     — SQL-миграции и сидеры
```

### Middleware

- **Recovery** — перехватывает паники в хендлерах, логирует через `slog`, возвращает 500
- **CORS** — разрешает запросы с любого origin (нужно для Swagger UI при удалённом деплое)

### Пул соединений PostgreSQL

Настроен в `internal/infrastructure/postgres/pool.go`:

| Параметр | Значение | Причина |
|----------|----------|---------|
| `MaxConns` | 20 | Ограничение нагрузки на БД |
| `MinConns` | 2 | Держать соединения тёплыми |
| `MaxConnLifetime` | 30 мин | Защита от stale-соединений |
| `MaxConnIdleTime` | 5 мин | Освобождение ресурсов при простое |
| `HealthCheckPeriod` | 1 мин | Проверка живости соединений |

---

## База данных

### Миграции

Миграции монтируются в `docker-entrypoint-initdb.d` и применяются автоматически при первом старте контейнера.

**`0001_create_tasks.up.sql`** — создание таблицы:

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id          BIGSERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_status     ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks (updated_at DESC);
```

Индекс на `updated_at DESC` добавлен для сортировки по дате обновления — типичный запрос в трекерах задач.

**`0002_add_recurrence_to_tasks.up.sql`** — добавление периодичности:

```sql
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS recurrence JSONB;

CREATE INDEX IF NOT EXISTS idx_tasks_recurrence_type
    ON tasks ((recurrence->>'type'))
    WHERE recurrence IS NOT NULL;
```

Индекс на `recurrence->>'type'` — частичный (только для строк с периодичностью), позволяет эффективно фильтровать задачи по типу повторения.

> При пересоздании БД: `docker compose down -v && docker compose up -d`

---

## API

Base URL: `/api/v1`  
Swagger UI: `/swagger/`

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/v1/tasks` | Список всех задач |
| `POST` | `/api/v1/tasks` | Создать задачу |
| `GET` | `/api/v1/tasks/{id}` | Получить задачу по ID |
| `PUT` | `/api/v1/tasks/{id}` | Обновить задачу |
| `DELETE` | `/api/v1/tasks/{id}` | Удалить задачу |

### Статусы задачи

`new` | `in_progress` | `done`

---

## Фича: периодичность задач

Задача может иметь опциональное поле `recurrence` с настройками повторения.

### Типы периодичности

#### `daily` — каждые N дней

```json
{
  "title": "Ежедневный обзвон пациентов",
  "recurrence": {
    "type": "daily",
    "day_interval": 2
  }
}
```

`day_interval` — целое число от 1 до 365.

#### `monthly` — в конкретные числа месяца

```json
{
  "title": "Формирование отчётности",
  "recurrence": {
    "type": "monthly",
    "month_days": [1, 15]
  }
}
```

`month_days` — непустой список чисел от 1 до 30. Дубли удаляются автоматически.

#### `specific_dates` — на конкретные даты

```json
{
  "title": "Плановая инвентаризация",
  "recurrence": {
    "type": "specific_dates",
    "specific_dates": ["2026-05-01", "2026-08-01", "2026-11-01"]
  }
}
```

Формат дат: `YYYY-MM-DD`. Дубли удаляются автоматически.

#### `even_odd_days` — чётные или нечётные числа месяца

```json
{
  "title": "Проверка журнала процедур",
  "recurrence": {
    "type": "even_odd_days",
    "even_odd_type": "even"
  }
}
```

`even_odd_type`: `"even"` или `"odd"`.

### Обновление периодичности (PUT)

Поле `recurrence` при обновлении имеет три состояния:

| Что передано | Результат |
|---|---|
| Поле отсутствует в JSON | Периодичность сохраняется без изменений |
| `"recurrence": null` | Периодичность удаляется |
| `"recurrence": {...}` | Периодичность заменяется новой |

### Хранение

Периодичность хранится как `JSONB`-поле в таблице `tasks`. Сериализация/десериализация выполняется в слое репозитория через `encoding/json`.

### Ошибки валидации (400 Bad Request)

| Ситуация | Сообщение |
|----------|-----------|
| Тип не указан | `invalid recurrence: type is required` |
| Неизвестный тип | `invalid recurrence: unknown recurrence type` |
| `daily` без `day_interval` | `invalid recurrence: day_interval is required for daily type` |
| `day_interval` вне [1, 365] | `invalid recurrence: day_interval must be between 1 and 365` |
| `monthly` без `month_days` | `invalid recurrence: month_days is required for monthly type` |
| Элемент `month_days` вне [1, 30] | `invalid recurrence: month_days values must be between 1 and 30` |
| `specific_dates` без дат | `invalid recurrence: specific_dates is required for specific_dates type` |
| Некорректная дата | `invalid recurrence: specific_dates contains invalid date` |
| `even_odd_days` без `even_odd_type` | `invalid recurrence: even_odd_type is required for even_odd_days type` |
| Недопустимое значение `even_odd_type` | `invalid recurrence: even_odd_type must be 'even' or 'odd'` |

---

## Принятые решения

### JSONB для хранения периодичности

Типы периодичности имеют разные наборы полей. Варианты хранения:

- **Отдельная таблица** — нормализованно, но требует JOIN на каждый запрос задачи
- **Отдельные колонки** — много NULL-полей, сложная схема
- **JSONB** — выбранный вариант: гибко, без JOIN, легко расширяется новыми типами

Компромисс: нет строгой типизации на уровне БД, но валидация выполняется в usecase-слое.

### `**RecurrenceSettings` в UpdateInput

`PUT` делает полную замену полей задачи. Для `recurrence` нужно различать три состояния: «не передано», «передано null», «передано значение». В Go `*RecurrenceSettings` не различает первые два случая при JSON-декодировании. Решение — double pointer `**RecurrenceSettings`: `nil` означает «не передано», `&nil` — «передан null», `&value` — «передано значение».

### Генерация OpenAPI из аннотаций

Изначально спека была написана вручную. Переход на `swaggo/swag` обеспечивает синхронизацию документации с кодом — спека генерируется при каждой сборке Docker-образа и при `make run` локально.

### Относительный URL сервера в OpenAPI (`"url": "/"`)

Фиксированный `localhost:8080` не работает при открытии Swagger UI с удалённого сервера. Относительный URL `/` заставляет Swagger UI делать запросы на тот же хост и порт, с которого открыта страница.

### Пул соединений PostgreSQL

Стандартные настройки `pgxpool` не ограничивают количество соединений явно. Добавлены явные лимиты (`MaxConns=20`, `MinConns=2`) и таймауты для предсказуемого поведения под нагрузкой и защиты от утечек соединений.

### Recovery middleware

Стандартный `net/http` не перехватывает паники — при панике в хендлере весь сервер падает. Recovery middleware логирует панику через `slog` и возвращает 500, не роняя сервис.

### Индекс на `updated_at DESC`

Добавлен в первую миграцию. Трекеры задач часто сортируют по дате последнего изменения — индекс делает такие запросы эффективными без полного скана таблицы.

---

## Использование LLM

Задание явно поощряет использование LLM-инструментов, поэтому указываю честно.

При разработке использовался **Kiro** (AI-powered IDE от Amazon) с моделью **Claude Sonnet** для:
- проектирования архитектуры фичи периодичности и обоснования технических решений (JSONB, double pointer, partial update)
- написания кода по слоям (domain → usecase → repository → transport)
- написания тестов в виде table-driven структур
- настройки Docker, переменных окружения и пула соединений
- автогенерации OpenAPI спеки через swaggo/swag (идея была предложена и реализована в процессе)
- написания миграций с индексами
- написания этого README

Все технические решения были обоснованы, проверены вручную и протестированы в реальном окружении. Код прошёл ревью, тесты запускались в Docker.
