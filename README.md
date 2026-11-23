# PhotoLab — Photo Editor

**PhotoLab** — веб-приложение для асинхронной обработки PNG-изображений. Фронтенд написан на **React + JavaScript**, API и обработчик изображений — на **Go**. Задачи передаются через **RabbitMQ**, пользователи и результаты сохраняются в **PostgreSQL**, сессии — в **Redis**, а метрики собирают **Prometheus и Grafana**.

## Возможности

- Регистрация, вход и авторизация через Bearer-токен.
- Загрузка PNG, выбор фильтра, сравнение до/после и скачивание готового изображения.
- Четыре фильтра: `Negative`, `Blur`, `Sharpen`, `FlipX`.
- Асинхронная очередь задач и статусы `in_progress`, `ready`, `failed`.
- Постоянное хранение пользователей, задач и результатов в PostgreSQL.
- Сессии в Redis с TTL **7 дней**.
- SQL-миграции, Swagger UI, Go- и Python-тесты, GitHub Actions CI.
- Мониторинг двух Go-приложений и дашборд Grafana с тремя графиками.

## Архитектура

```text
                        PhotoLab (React/Vite :5173)
                                  |
                            Vite /api proxy
                                  |
                                  v
                           Go HTTP API :8001
                             /     |      \
                            /      |       \
                           v       v        v
                     PostgreSQL   Redis   RabbitMQ
                      users/tasks sessions   |
                                             v
                                    ImageProcessor (Go)
                                             |
                                      POST /commit
                                             |
                                             v
                                        Go HTTP API

               Go HTTP API :8001/metrics  ─┐
           ImageProcessor :9091/metrics  ───┼─> Prometheus :9090
                                          └────────> Grafana :3000
```

1. Пользователь отправляет PNG и выбранный фильтр в API.
2. Go сохраняет задачу со статусом `in_progress` в PostgreSQL и публикует сообщение в RabbitMQ.
3. Отдельный ImageProcessor получает сообщение, обрабатывает изображение и отправляет результат в служебный `POST /commit`.
4. API записывает статус и результат в PostgreSQL; фронтенд периодически запрашивает статус и получает готовую PNG.
5. Prometheus опрашивает `/metrics` у обоих Go-процессов, а Grafana строит графики.

## Технологии

| Компонент | Технология |
| --- | --- |
| Фронтенд | React, JavaScript (JSX), Vite |
| API | Go, `net/http`, Swagger / Swaggo |
| Обработка изображений | Go, `imaging` |
| Брокер сообщений | RabbitMQ, `amqp091-go` |
| Постоянное хранилище | PostgreSQL, `pgx/v5/pgxpool` |
| Сессии | Redis, `go-redis/v9` |
| Миграции | `rubenv/sql-migrate` |
| Метрики | `prometheus/client_golang`, Prometheus |
| Дашборды | Grafana |
| Тесты и CI | Go testing, pytest, Pillow, GitHub Actions |
| Инфраструктура | Docker Compose |

## Быстрый запуск

### Требования

- Docker и Docker Compose (`docker compose`).
- Node.js и npm — для локального запуска React-фронтенда.
- Go — для разработки, Go-тестов и запуска `sql-migrate`.
- Python 3 — для интеграционных тестов.

```bash
# Получить репозиторий
 git clone https://github.com/Daisy32411/photo_editor.git
 cd photo_editor

# Запустить сервер, Processor, брокер, базы данных и мониторинг
 docker compose up -d --build
 docker compose ps
```

> **Важно:** на совершенно новой базе нужно **применить миграции**, прежде чем регистрировать пользователей или запускать обработку. `docker compose up` сам по себе не создаёт таблицы `users` и `tasks`.

Установить CLI миграций (один раз):

```bash
go install github.com/rubenv/sql-migrate/...@latest
```

В репозитории есть `dbconfig.yml` с настройками подключения с хост-машины: **`localhost:5430`**. Применить миграции из корня проекта:

```bash
"$(go env GOPATH)/bin/sql-migrate" up
"$(go env GOPATH)/bin/sql-migrate" status
```

Затем запустить фронтенд **в отдельном терминале**:

```bash
cd frontend
npm install
npm run dev
```

Открой **http://localhost:5173**. Vite проксирует запросы `/api/*` на `http://127.0.0.1:8001/*`, поэтому при локальной разработке отдельно настраивать CORS не нужно. Фронтенд пока запускается **вне Docker Compose**.

### Адреса сервисов

| Сервис | Локальный адрес |
| --- | --- |
| PhotoLab (React) | http://localhost:5173 |
| Go API | http://localhost:8001 |
| Swagger UI | http://localhost:8001/swagger/index.html |
| RabbitMQ Management | http://localhost:15672 |
| PostgreSQL | `localhost:5430` |
| Processor metrics | http://localhost:9091/metrics |
| Go API metrics | http://localhost:8001/metrics |
| Prometheus | http://localhost:9090 |
| Prometheus targets | http://localhost:9090/targets |
| Grafana | http://localhost:3000 |

Локальные учётные данные: RabbitMQ — `photo` / `photo`; PostgreSQL — `photo` / `photo` (база `photo_editor`); Grafana — `admin` / `photolab_dev`.

Эти данные предназначены **только для локальной разработки**. Не используйте их для публичного развёртывания.

### Полезные команды

```bash
make up                 # собрать и запустить Docker-сервисы
make down               # остановить Docker-сервисы
make ps                 # состояние контейнеров
make logs               # поток логов
make check              # go vet + go test
make build              # собрать server и processor в bin/

docker compose logs -f server processor
```

**Не выполняйте `docker compose down -v`, если хотите сохранить данные**: ключ `-v` удаляет Docker volumes, включая PostgreSQL, Redis, Prometheus и Grafana.

## REST API

Все пользовательские запросы, кроме регистрации и входа, требуют заголовок:

```http
Authorization: Bearer <token>
```

| Метод | Маршрут | Назначение |
| --- | --- | --- |
| `POST` | `/register` | Регистрация пользователя |
| `POST` | `/login` | Вход и получение токена |
| `POST` | `/task` | Создать задачу обработки PNG |
| `GET` | `/status/{id}` | Получить статус задачи |
| `GET` | `/result/{id}` | Скачать обработанную PNG |
| `POST` | `/commit` | Служебный приём результата от Processor |
| `GET` | `/metrics` | Метрики Go API |

### Регистрация

```bash
curl -i -X POST http://localhost:8001/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alex","password":"123456"}'
```

Успех — HTTP `201 Created`; повторная регистрация того же имени — `409 Conflict`.

### Вход

```bash
curl -X POST http://localhost:8001/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alex","password":"123456"}'
```

Пример ответа:

```json
{"token":"<64-символьный-hex-токен>"}
```

Сессия хранится в Redis под ключом `session:<token>` с TTL семь дней.

### Создание задачи

`POST /task` принимает PNG в Base64 и настройки фильтра:

```json
{
  "image": "<base64-encoded-png>",
  "filter": {
    "name": "Blur",
    "parameters": {
      "sigma": 2.5
    }
  }
}
```

Ответ при успехе — HTTP `201 Created`:

```json
{"task_id":"<uuid>"}
```

### Статус и результат

```bash
curl http://localhost:8001/status/<uuid> \
  -H 'Authorization: Bearer <token>'
```

Ответ:

```json
{"status":"in_progress"}
```

Возможные статусы: `in_progress`, `ready`, `failed`.

Когда статус `ready`, результат доступен как **сырые PNG-байты**, не JSON:

```bash
curl -fS http://localhost:8001/result/<uuid> \
  -H 'Authorization: Bearer <token>' \
  -o result.png
```

Пока результат не готов, `/result/{id}` возвращает `409 Conflict`. При отсутствии задачи возвращается `404 Not Found`.

### Фильтры

| Фильтр | Действие | Параметры |
| --- | --- | --- |
| `Negative` | Инверсия RGB с сохранением прозрачности | Нет |
| `Blur` | Размытие | `sigma`: `0 < sigma <= 20` |
| `Sharpen` | Повышение резкости | `sigma`: `0 < sigma <= 20` |
| `FlipX` | Вертикальное отражение (относительно оси X) | Нет |

## PostgreSQL и Redis

### Миграции

Схема PostgreSQL хранится в директории `migrations/`:

```text
migrations/
├── 001_create_users.sql
└── 002_create_tasks.sql
```

- `users`: `id`, `username` (уникальный), `password_hash`, `created_at`.
- `tasks`: `id`, `status`, `result`, `created_at`.
- `result` в текущей реализации — PNG, закодированная в Base64, в поле `TEXT`.
- `schema_migrations` используется инструментом `sql-migrate` для учёта применённых изменений.

Посмотреть задачи без вывода больших Base64-строк:

```bash
docker compose exec postgres psql -U photo -d photo_editor \
  -c 'SELECT id, status, LENGTH(result) AS result_size, created_at FROM tasks ORDER BY created_at DESC LIMIT 5;'
```

Посмотреть пользователей:

```bash
docker compose exec postgres psql -U photo -d photo_editor \
  -c 'SELECT id, username FROM users;'
```

### Сессии Redis

Сессии сохраняются в Redis с TTL **604800 секунд (7 дней)**. Сохраняется связь токена с UUID пользователя; пароль и bcrypt-хеш в Redis не передаются.

```bash
docker compose exec redis redis-cli ping
docker compose exec redis redis-cli --scan --pattern 'session:*'
```

После перезапуска **только** Go-сервера действующие сессии остаются доступными, если срок их действия не истёк и Redis сохранил данные.

## Мониторинг: Prometheus + Grafana

Go API экспортирует стандартные runtime-метрики по `/metrics` на порту `8001`. ImageProcessor экспортирует свои метрики по `/metrics` на порту `9091`.

**Метрики ImageProcessor:**

| Метрика | Тип | Значение |
| --- | --- | --- |
| `processor_filter_usage_total{filter}` | Counter | Количество попыток применения каждого фильтра |
| `processor_tasks_total{filter,status}` | Counter | Количество успешных и неуспешных обработок |
| `processor_processing_duration_seconds{filter}` | Histogram | Длительность обработки изображения в секундах |

Время измеряется вокруг обработки PNG (`processImage`), а не вокруг ожидания HTTP-ответа от `/commit`. Повторная доставка сообщения RabbitMQ считается отдельной попыткой.

Настройки Prometheus: `monitoring/prometheus.yml`. Он опрашивает `server:8001` и `processor:9091` каждые **5 секунд**.

Проверить состояние целей: **http://localhost:9090/targets**. Обе должны иметь статус `UP`.

### Графики Grafana

Дашборд **PhotoLab — Monitoring** включает:

1. **Использование фильтров** — количество запусков каждого фильтра.
2. **Среднее время обработки** — средняя длительность для каждого фильтра.
3. **Результаты обработки** — успешные и неуспешные попытки.

Примеры запросов PromQL:

```promql
sum by (filter) (processor_filter_usage_total)
```

```promql
sum by (filter) (processor_processing_duration_seconds_sum)
/
sum by (filter) (processor_processing_duration_seconds_count)
```

```promql
sum by (status) (processor_tasks_total)
```

Настройки хранятся в:

```text
monitoring/
├── prometheus.yml
└── grafana/
    ├── datasources.yml
    ├── dashboards.yml
    └── dashboards/
        └── photolab.json
```

Grafana автоматически подключает Prometheus как источник данных и загружает дашборд из JSON-файла при запуске контейнера.

> Метрики с метками отдельных фильтров появляются после первой обработки соответствующих задач. Счётчики конкретного процесса обнуляются после его перезапуска; Prometheus сохраняет собранную ранее историю в `prometheus_data`.

## Тестирование

### Go

```bash
go test ./...
go vet ./...
```

Тесты фильтров находятся в `internal/processor/filters_test.go`.

### Интеграционные тесты PostgreSQL

Тест репозитория задач использует **отдельную базу** `photo_editor_test` и переменную `POSTGRES_TEST_URL`.

```bash
docker compose exec postgres createdb -U photo photo_editor_test
```

Добавь в `dbconfig.yml` окружение `test`, если его ещё нет:

```yaml
test:
  dialect: postgres
  datasource: "host=localhost port=5430 user=photo password=photo dbname=photo_editor_test sslmode=disable"
  dir: migrations
  table: schema_migrations
```

Примени миграции тестовой базы и запусти проверки:

```bash
"$(go env GOPATH)/bin/sql-migrate" up -env=test
POSTGRES_TEST_URL='postgres://photo:photo@localhost:5430/photo_editor_test?sslmode=disable' \
  go test -v ./internal/storage
```

Если `POSTGRES_TEST_URL` не задан, тесты PostgreSQL могут быть пропущены (`SKIP`). Если база уже существует, повторно создавать её не требуется.

### Python: полный путь обработки

Интеграционный тест `test.py` проверяет регистрацию, вход, отправку задачи, ожидание статуса и получение PNG для четырёх фильтров. Требуются запущенные сервисы и применённые миграции.

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install pytest requests Pillow
python -m pytest -v test.py
```

Запускать команды нужно **из корня `photo_editor`**.

### GitHub Actions

Workflow: `.github/workflows/ci.yml`. CI проверяет Go-код, собирает оба приложения, запускает Docker Compose, применяет SQL-миграции и выполняет интеграционные тесты Python. При ошибке можно посмотреть логи контейнеров в GitHub Actions.

## Структура проекта

```text
photo_editor/
├── cmd/
│   ├── server/               # Go HTTP API
│   └── processor/            # RabbitMQ consumer + HTTP /metrics
├── internal/
│   ├── broker/               # RabbitMQ publisher
│   ├── handlers/             # HTTP-хендлеры и авторизация
│   ├── models/               # модели и ошибки приложения
│   ├── processor/            # обработка изображений и метрики
│   ├── service/              # бизнес-логика и интерфейсы
│   └── storage/              # PostgreSQL, Redis
├── frontend/                 # React + JavaScript + Vite
├── migrations/               # SQL-миграции PostgreSQL
├── monitoring/               # Prometheus и Grafana
├── docs/                     # Swagger
├── .github/workflows/ci.yml
├── docker-compose.yml
├── dbconfig.yml
├── Dockerfile.server
├── Dockerfile.processor
├── Makefile
├── go.mod
└── test.py
```

## Ограничения и безопасность

- Проект настроен для **локальной разработки**. Логины/пароли в Docker Compose демонстрационные; перед публикацией необходимо заменить их и использовать секреты/переменные окружения.
- `POST /commit` — **внутренний служебный маршрут**. В текущей версии у него нет отдельной авторизации: для публичного развёртывания необходимо ограничить доступ или защитить его.
- `/metrics`, PostgreSQL, RabbitMQ Management и Grafana нельзя открывать в интернет без соответствующей защиты и ограничений доступа.
- Результаты PNG хранятся в PostgreSQL как Base64 `TEXT`; для больших объёмов разумнее использовать файловое/object storage, сохранив в БД ссылку.
- Состояние выполнения задачи хранится в PostgreSQL, но обработка через RabbitMQ и сохранение в БД не объединены в одну транзакцию. Для production-проекта потребуется продумать повторные попытки, согласованность операций и обработку зависших задач.
- RabbitMQ ограничивает размер сообщения через `rabbitmq.conf` (`max_message_size`); Base64 увеличивает объём исходного PNG примерно на треть. Сервер также проверяет `MaxTaskMessageSize`.
- Состояние сессии в Redis ограничено TTL; по истечении срока нужно выполнить вход заново.

---

**PhotoLab** — учебный full-stack проект, объединяющий Go, React, асинхронную обработку изображений, постоянное хранение данных, тестирование и мониторинг.