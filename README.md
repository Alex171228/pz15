# Notes API — Практическое занятие №14 (PostgreSQL, pool, оптимизация запросов)

Этот проект — минимальный REST API на Go для работы с заметками (CRUD) с хранением данных в PostgreSQL.
Сделан под требования ПЗ-14: connection pool, индексы, keyset-пагинация, батчинг ANY($1), prepared statements,
транзакции, команды для EXPLAIN/ANALYZE и нагрузочных тестов.

## Структура проекта

```text
notes-api-pz14/
  cmd/api/main.go            # запуск HTTP сервера
  internal/config/config.go  # чтение env
  internal/db/db.go          # подключение к PostgreSQL + pool
  internal/notes/            # модели, repository (SQL), handlers, маршруты
  migrations/001_init.sql    # таблицы + индексы
  scripts/explain.sql        # примеры EXPLAIN (ANALYZE, BUFFERS)
  .env.example               # пример переменных окружения
```

## 1) Подготовка PostgreSQL (на сервере)

### 1.1 Создать БД и пользователя

```bash
sudo -u postgres psql
```

```sql
CREATE USER notes_user WITH PASSWORD 'notes_pass';
CREATE DATABASE notes_db OWNER notes_user;
GRANT ALL PRIVILEGES ON DATABASE notes_db TO notes_user;
\q
```

### 1.2 Применить миграции

```bash
psql "postgres://notes_user:notes_pass@localhost:5432/notes_db?sslmode=disable" -f migrations/001_init.sql
```

## 2) Настройка переменных окружения

Скопируй `.env.example` и выставь значения (или экспортируй вручную):

```bash
export DATABASE_URL="postgres://notes_user:notes_pass@localhost:5432/notes_db?sslmode=disable"

# pool (примерные стартовые значения)
export DB_MAX_OPEN=20
export DB_MAX_IDLE=10
export DB_CONN_MAX_LIFETIME=30m
export DB_CONN_MAX_IDLE_TIME=5m

export HTTP_ADDR=":8080"
```

## 3) Запуск сервера

В корне проекта:

```bash
go mod tidy
go run ./cmd/api
```

Проверка:

- Health: `GET http://<IP>:8080/health`
- API ниже.

## 4) REST API

### 4.1 Создать заметку

`POST /notes`

Body:
```json
{ "title": "Hello", "content": "world" }
```

### 4.2 Получить заметку

`GET /notes/{id}`

### 4.3 Обновить заметку

`PUT /notes/{id}`

Body:
```json
{ "title": "New title", "content": "New content" }
```

### 4.4 Удалить заметку

`DELETE /notes/{id}`

### 4.5 Список заметок (keyset pagination)

`GET /notes?limit=20` — первая страница

Следующая страница использует курсор последней записи предыдущей страницы:

`GET /notes?limit=20&cursor_created_at=2025-12-21T10:00:00Z&cursor_id=123`

Сортировка: `created_at DESC, id DESC`.

### 4.6 Поиск по заголовку (GIN индекс)

`GET /notes?limit=20&q=redis`

Поиск реализован через `to_tsvector(title) @@ plainto_tsquery(...)` и индекс GIN.

### 4.7 Батч получение по id (ANY($1))

`POST /notes/batch`

Body:
```json
{ "ids": [1,2,3] }
```

## 5) Оптимизации, реализованные в проекте

1. **Connection pool** (`database/sql`):
   - `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`, `SetConnMaxIdleTime` — задаются через env.
2. **Keyset pagination** вместо OFFSET: `WHERE (created_at, id) < ($1,$2)`.
3. **Batching** вместо N+1: `WHERE id = ANY($1)`.
4. **Prepared statements**: INSERT/SELECT/UPDATE/DELETE готовятся через `PrepareContext` при старте репозитория.
5. **Транзакция** при создании заметки: INSERT в `notes` + запись в `notes_audit` в одной транзакции.

## 6) EXPLAIN / ANALYZE (для отчёта)

Примеры запросов и команд — в `scripts/explain.sql`.

Запуск (на сервере):

```bash
psql "$DATABASE_URL" -f scripts/explain.sql
```

## 7) Нагрузочные тесты (до/после, RPS, p95/p99)

Пример с `hey` (на твоём локальном ПК):

```bash
hey -n 2000 -c 50 "http://<IP>:8080/notes?limit=20"
hey -n 2000 -c 50 "http://<IP>:8080/notes/1"
```

Сравни результаты при разных настройках пула (`DB_MAX_OPEN`, `DB_MAX_IDLE`) и при переключении пагинации (OFFSET vs keyset — см. `scripts/explain.sql`).

---

## Быстрый seed данных (если надо)

```sql
INSERT INTO notes(title, content)
SELECT 'title ' || gs, 'content ' || gs
FROM generate_series(1, 50000) gs;
```
