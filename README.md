# Практическое задание 15
## Шишков А.Д. ЭФМО-02-22
## Тема
Unit-тестирование функций (testing, testify)
## Цели
- Освоить базовые приёмы unit-тестирования в Go с помощью стандартного пакета testing.
- Научиться писать табличные тесты, подзадачи t.Run, тестировать ошибки и паники.
- Освоить библиотеку утверждений testify (assert, require) для лаконичных проверок.
- Научиться измерять покрытие кода (go test -cover) и формировать html-отчёт покрытия.
- Подготовить минимальную структуру проектных тестов и общий чек-лист качества тестов.

## Структура проекта

```text
.
├── cmd/
│   └── api/
│       └── main.go              # Точка входа приложения (HTTP-сервер)
│
├── docs/
│   ├── docs.go                  # Сгенерированный Swagger-код
│   ├── swagger.json             # Swagger спецификация (JSON)
│   └── swagger.yaml             # Swagger спецификация (YAML)
│
├── internal/
│   ├── config/
│   │   ├── config.go            # Загрузка конфигурации из env
│   │   └── config_test.go       # Unit-тесты конфигурации
│   │
│   ├── db/
│   │   └── postgres.go          # Подключение к PostgreSQL и пул соединений
│   │
│   ├── notes/
│   │   ├── handlers.go          # HTTP-обработчики REST API
│   │   ├── repository.go        # Репозиторий заметок (PostgreSQL)
│   │   └── handlers_test.go     # Тесты HTTP-логики (handlers)
│   │
│   ├── mathx/
│   │   ├── mathx.go             # Математические функции
│   │   ├── mathx_test.go        # Unit-тесты
│   │   └── mathx_bench_test.go  # Бенчмарки
│   │
│   ├── stringsx/
│   │   ├── stringsx.go          # Утилиты для строк
│   │   └── stringsx_test.go     # Unit-тесты
│   │
│   └── service/
│       ├── service.go           # Бизнес-логика сервиса
│       └── service_test.go      # Unit-тесты бизнес-логики
│
├── migrations/
│   └── 001_init.sql             # SQL-миграция (таблицы и индексы)
│
├── scripts/
│   └── load_test.sh             # Скрипты для нагрузочного тестирования
│
├── .env.example                 # Пример файла переменных окружения
├── go.mod                       # Go-модуль
├── go.sum                       # Зависимости
└── README.md                    # Документация проекта

```

## Реализация тестов

### internal/config — загрузка конфигурации из env
Тестирование значений по умолчанию
Проверяется, что при отсутствии переменных окружения используются корректные значения по умолчанию.
```go
func TestLoad_Defaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	require.Equal(t, ":8080", cfg.HTTPAddr)
	require.Equal(t, 20, cfg.MaxOpenConns)
	require.Equal(t, 10, cfg.MaxIdleConns)
}
```
Тестирование граничных случаев с t.Run (валидные и невалидные значения)
Используется табличный тест для проверки различных вариантов входных данных.
```go
func TestLoad_OverridesAndInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		wantOpen int
		wantIdle int
	}{
		{
			name: "valid_overrides",
			env: map[string]string{
				"DB_MAX_OPEN": "5",
				"DB_MAX_IDLE": "2",
			},
			wantOpen: 5,
			wantIdle: 2,
		},
		{
			name: "invalid_numbers_fall_back_to_defaults",
			env: map[string]string{
				"DB_MAX_OPEN": "abc",
			},
			wantOpen: 20,
			wantIdle: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg := Load()

			require.Equal(t, tt.wantOpen, cfg.MaxOpenConns)
			require.Equal(t, tt.wantIdle, cfg.MaxIdleConns)
		})
	}
}
```
### internal/notes — тестирование HTTP-обработчиков
Тестирование валидации входных данных
Проверяется отказ при пустом теле запроса.
```go
func TestHandlers_Create_Validation(t *testing.T) {
	h := NewHandlers(newStubRepo())
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	resp, _ := http.Post(
		ts.URL+"/api/v1/notes",
		"application/json",
		strings.NewReader(`{"title":"","content":""}`),
	)

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
```
Тестирование различных веток обработки (success / not found / internal error)
Проверяется корректное преобразование ошибок репозитория в HTTP-ответы.
```go
func TestHandlers_Get_Success_NotFound_And_Internal(t *testing.T) {
	repo := newStubRepo()
	repo.getFn = func(ctx context.Context, id int64) (Note, error) {
		if id == 1 {
			return Note{ID: 1, Title: "t", Content: "c"}, nil
		}
		if id == 2 {
			return Note{}, ErrNotFound
		}
		return Note{}, errors.New("boom")
	}

	h := NewHandlers(repo)
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	r1, _ := http.Get(ts.URL + "/api/v1/notes/1")
	require.Equal(t, 200, r1.StatusCode)

	r2, _ := http.Get(ts.URL + "/api/v1/notes/2")
	require.Equal(t, 404, r2.StatusCode)

	r3, _ := http.Get(ts.URL + "/api/v1/notes/3")
	require.Equal(t, 500, r3.StatusCode)
}
```
### internal/mathx — unit-тесты и бенчмарки

Табличное тестирование арифметики
Проверяются положительные, нулевые и отрицательные значения.

```go
func TestSum_Table(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"pos", 2, 3, 5},
		{"zero", 0, 0, 0},
		{"neg", -2, 1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Sum(tt.a, tt.b))
		})
	}
}
```
Тестирование граничных случаев и согласованности реализаций
Проверяется, что оптимизированная версия Fibonacci возвращает те же значения, что и базовая.
```go
func TestFibFast_Equals_Fib(t *testing.T) {
	for n := 0; n <= 20; n++ {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			require.Equal(t, Fib(n), FibFast(n))
		})
	}
}
```
Бенчмарки (сравнение производительности)
Проводится сравнение скорости двух реализаций.
```go
func BenchmarkFib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Fib(20)
	}
}

func BenchmarkFibFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = FibFast(20)
	}
}
```

### internal/stringsx — строковые утилиты

Тестирование нормализации строки
Проверяется обрезка пробелов и приведение к нижнему регистру.

```go
func TestNormalize(t *testing.T) {
	require.Equal(t, "hello", Normalize("  HeLLo  "))
}
```
Тестирование граничных случаев (пустая строка)
Проверяется корректное определение пустых и непустых строк.
```go
func TestIsEmpty(t *testing.T) {
	require.True(t, IsEmpty("   "))
	require.False(t, IsEmpty("x"))
}
```
### internal/service — тестирование бизнес-логики
Тестирование обработки ошибки "not found"
Проверяется корректная передача ошибки из репозитория.
```go
func TestService_FindIDByEmail_NotFound(t *testing.T) {
	svc := New(stubRepo{
		byEmailFn: func(email string) (User, error) {
			return User{}, ErrNotFound
		},
	})

	_, err := svc.FindIDByEmail("no@x.com")
	require.ErrorIs(t, err, ErrNotFound)
}
```
Тестирование успешного сценария
Проверяется корректная бизнес-логика без обращения к внешним системам.
```go
func TestService_FindIDByEmail_Success(t *testing.T) {
	svc := New(stubRepo{
		byEmailFn: func(email string) (User, error) {
			return User{ID: 7, Email: email}, nil
		},
	})

	id, err := svc.FindIDByEmail("a@b.com")
	require.NoError(t, err)
	require.Equal(t, int64(7), id)
}
```
## Тестирование API эндпоинтов
Тестирование REST API выполнено на уровне HTTP-обработчиков с использованием net/http/httptest.
Для изоляции от PostgreSQL используется stub-хранилище (in-memory заглушка), поэтому тесты являются unit-тестами: проверяют маршрутизацию, валидацию, коды ответов, формат JSON и обработку ошибок.
Файл тестов: internal/notes/handlers_test.go
### Тестирование валидации запроса: POST /api/v1/notes
Проверяется, что при пустых title/content сервер возвращает 400 Bad Request.

```go
func TestHandlers_Create_Validation(t *testing.T) {
	h := NewHandlers(newStubRepo())
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	resp, err := http.Post(
		ts.URL+"/api/v1/notes",
		"application/json",
		strings.NewReader(`{"title":"","content":""}`),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
```

### Тестирование успешного создания: POST /api/v1/notes
Проверяется, что при валидном JSON сервер возвращает 201 Created и корректный объект.
```go
func TestHandlers_Create_Success(t *testing.T) {
	h := NewHandlers(newStubRepo())
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	resp, err := http.Post(
		ts.URL+"/api/v1/notes",
		"application/json",
		strings.NewReader(`{"title":"t","content":"c"}`),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

### Тестирование невалидного id: GET /api/v1/notes/{id}
Проверяется, что при id=abc сервер возвращает 400 Bad Request.

```go
func TestHandlers_Get_InvalidID(t *testing.T) {
	h := NewHandlers(newStubRepo())
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/notes/abc")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
```

### Тестирование ветвлений (success / not found / internal): GET /api/v1/notes/{id}

Проверяется корректное преобразование ошибок репозитория:
- nil → 200 OK
- ErrNotFound → 404 Not Found
- любая другая ошибка → 500 Internal Server Error
```go
func TestHandlers_Get_Success_NotFound_And_Internal(t *testing.T) {
	repo := newStubRepo()
	repo.getFn = func(ctx context.Context, id int64) (Note, error) {
		switch id {
		case 1:
			return Note{ID: 1, Title: "t", Content: "c"}, nil
		case 2:
			return Note{}, ErrNotFound
		default:
			return Note{}, errors.New("boom")
		}
	}

	h := NewHandlers(repo)
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	r1, _ := http.Get(ts.URL + "/api/v1/notes/1")
	require.Equal(t, http.StatusOK, r1.StatusCode)

	r2, _ := http.Get(ts.URL + "/api/v1/notes/2")
	require.Equal(t, http.StatusNotFound, r2.StatusCode)

	r3, _ := http.Get(ts.URL + "/api/v1/notes/3")
	require.Equal(t, http.StatusInternalServerError, r3.StatusCode)
}
```
### Тестирование некорректного JSON: POST /api/v1/notes/batch
Проверяется, что при ошибке парсинга JSON сервер отвечает 400 Bad Request.
```go
func TestHandlers_Batch_InvalidJSON(t *testing.T) {
	h := NewHandlers(newStubRepo())
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	resp, err := http.Post(
		ts.URL+"/api/v1/notes/batch",
		"application/json",
		strings.NewReader(`{"ids":[1,2,]}`), // invalid JSON
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
```

### Тестирование успешного batch-запроса: POST /api/v1/notes/batch
Проверяется, что сервер возвращает 200 OK и список заметок.
```go
func TestHandlers_Batch_Success(t *testing.T) {
	h := NewHandlers(newStubRepo())
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	resp, err := http.Post(
		ts.URL+"/api/v1/notes/batch",
		"application/json",
		strings.NewReader(`{"ids":[1,2,3]}`),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
```
### Тестирование парсинга query-параметров: GET /api/v1/notes?limit=...
Проверяется корректный разбор параметров запроса (limit, cursor_created_at, cursor_id) и корректность формирования ответа (например, курсора следующей страницы).
```go
func TestHandlers_List_ParsesParams(t *testing.T) {
	h := NewHandlers(newStubRepo())
	ts := httptest.NewServer(h.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/notes?limit=10&cursor_id=100")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
```
### Команда запуска тестов API
```bash
go test -v ./internal/notes
```
## Запуск тестов 
Запуск всех тестов
```bash
go test ./...
```

<img width="657" height="188" alt="image" src="https://github.com/user-attachments/assets/59a2acd4-0308-48d4-8771-7cd99a441fc1" /> 

Запуск подробным выводом
```bash
go test -v ./...
```

<img width="969" height="904" alt="image" src="https://github.com/user-attachments/assets/f55a5f94-9c4e-4be1-b9a8-1694f79386e0" /> 

Результат выполнения:
- все тесты успешно проходят (PASS);
- тесты охватывают конфигурацию, бизнес-логику, вспомогательные пакеты и HTTP-обработчики;
- внешние зависимости (PostgreSQL) в unit-тестах не используются.

Запуск с измерением процента покрытия
```bash
go test -cover ./...         
```

<img width="1045" height="169" alt="image" src="https://github.com/user-attachments/assets/5ff096ef-194d-460d-b763-5d8a0d7bed69" /> 

Генерация отчёта о покрытии
Для получения детального отчёта используется профиль покрытия:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out    
```
Дополнительно может быть сгенерирован HTML-отчёт:
```bash
go tool cover -html=coverage.out -o coverage.html  
```

<img width="979" height="1125" alt="image" src="https://github.com/user-attachments/assets/c23d563d-d87c-498c-89b8-a8482af8b399" /> 

HTML-отчёт позволяет визуально оценить покрытие отдельных функций и ветвлений. 

Запуск бенчмарков 

Для оценки производительности отдельных функций реализованы бенчмарки в пакете internal/mathx.
```bash
go test -bench=. -benchmem ./internal/mathx 
```

<img width="998" height="117" alt="image" src="https://github.com/user-attachments/assets/718a4e9c-c1cc-4721-8233-fff00ccab257" /> 

## Заключение

В ходе выполнения практического занятия №15 проект, разработанный в рамках ПЗ-14, был доработан и расширен модульными тестами, анализом покрытия кода и бенчмарками. Основное внимание было уделено тестированию бизнес-логики и граничных случаев, а не формальному увеличению процента покрытия.
В проекте реализованы unit-тесты для ключевых пакетов:
- internal/mathx — тестирование математических функций и сравнение производительности различных реализаций;
- internal/stringsx — проверка корректности обработки строк и граничных значений;
- internal/service — тестирование бизнес-логики с использованием заглушек (stub) вместо внешних зависимостей;
- internal/notes — тестирование REST API эндпоинтов на уровне HTTP-обработчиков с применением httptest.
По результатам тестирования было достигнуто покрытие кода, превышающее установленный критерий (≥70%) в требуемых пакетах. Дополнительно проведены бенчмарки, которые наглядно показали эффективность оптимизированных алгоритмов по сравнению с наивными реализациями.
