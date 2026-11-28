# PhotoLab — Photo Editor

**PhotoLab** — учебное веб-приложение для асинхронной обработки PNG-изображений. Фронтенд написан на React + JavaScript, HTTP API и отдельный ImageProcessor — на Go. Задачи передаются через RabbitMQ, пользователи и результаты сохраняются в PostgreSQL, сессии — в Redis. Мониторинг реализован с помощью Prometheus и Grafana.

## Возможности

- Регистрация, вход, bcrypt-хеширование паролей и авторизация через Bearer-токен.
- Загрузка PNG, выбор фильтра, сравнение до/после и скачивание результата.
- Четыре фильтра: `Negative`, `Blur`, `Sharpen`, `FlipX`.
- Асинхронное выполнение задач со статусами `in_progress`, `ready`, `failed`.
- Привязка задач к пользователям и проверка владельца при чтении статуса и результата.
- Служебная авторизация Processor, атомарное завершение задач и идемпотентность повторных commit-запросов.
- RabbitMQ Publisher Confirms, восстановление соединения, ограниченные повторные доставки и Dead Letter Queue (DLQ).
- Ограничения размера HTTP-запроса, AMQP-сообщения и количества пикселей PNG.
- PostgreSQL, Redis с TTL сессий 7 дней, SQL-миграции, Swagger, Go- и Python-тесты, GitHub Actions CI.
- Метрики двух Go-приложений и автоматически загружаемый дашборд Grafana.

## Архитектура

```text
                   PhotoLab (React/Vite :5173)
                              |
                         Vite /api proxy
                              |
                       Go HTTP API :8001
                        /      |       \
                       v       v        v
                 PostgreSQL   Redis   RabbitMQ
                 users/tasks sessions image_tasks (quorum)
                                          |
                                          v
                                     ImageProcessor
                                          |
                               POST /commit + processor token
                                          |
                                          v
                                     Go HTTP API

                 RabbitMQ image_tasks --> image_tasks_dlx --> image_tasks_dlq
                   (rejected / delivery limit)

                 Go API :8001/metrics --------+
                 Processor :9091/metrics -----+--> Prometheus :9090
                                                          ^
                                                          |
                                                    Grafana :3000
```

1. Пользователь отправляет PNG в Base64 и выбранный фильтр в API.
2. API проверяет авторизацию и лимиты, сохраняет задачу с `user_id` и статусом `in_progress`, публикует сообщение в RabbitMQ с подтверждением брокера.
3. ImageProcessor получает сообщение, проверяет PNG и количество пикселей, применяет фильтр и отправляет результат в `POST /commit` с `X-Processor-Token`.
4. API атомарно переводит задачу в конечный статус. Processor подтверждает сообщение (`ACK`) после успешного commit. Фронтенд запрашивает статус и скачивает результат.
5. Постоянные ошибки отправляются в DLQ; временные ошибки допускают ограниченные повторные доставки. Prometheus собирает метрики, Grafana визуализирует их.

## Технологии

| Компонент | Технология |
| --- | --- |
| Фронтенд | React, JavaScript (JSX), Vite |
| API | Go, `net/http`, Swagger / Swaggo |
| Обработка изображений | Go, `imaging` |
| Брокер | RabbitMQ 4, `amqp091-go` |
| Постоянное хранилище | PostgreSQL 17, `pgx/v5/pgxpool` |
| Сессии | Redis 7, `go-redis/v9` |
| Миграции | `rubenv/sql-migrate` |
| Метрики | `prometheus/client_golang`, Prometheus |
| Дашборды | Grafana |
| Тесты и CI | Go testing, pytest, requests, Pillow, Pika, GitHub Actions |
| Инфраструктура | Docker Compose, Make |

## Быстрый запуск

### Требования

- Docker и Docker Compose v2, Make, OpenSSL, curl.
- Go (версия указана в `go.mod`) — для миграций и локальных Go-тестов.
- Node.js и npm — для React-фронтенда.
- Python 3.12 — для интеграционных тестов.

Получить репозиторий:

```bash
git clone -b hw1 https://github.com/Daisy32411/photo_editor.git
cd photo_editor
```

### Секрет Processor

Создать локальный `.env` (один раз):

```bash
touch .env
chmod 600 .env
grep -q '^PROCESSOR_TOKEN=' .env || \
  printf '\nPROCESSOR_TOKEN=%s\n' "$(openssl rand -hex 32)" >> .env
```

`PROCESSOR_TOKEN` должен быть непустым и одинаковым для Server и Processor. Docker Compose читает `.env` автоматически. Если в терминале уже экспортирован другой токен, он имеет приоритет над `.env`; используйте одно и то же значение для контейнеров и тестов.

Не коммитьте `.env`. Он должен быть исключён из Git (`.gitignore`) и из контекста Docker-сборки (`.dockerignore`). В CI токен генерируется автоматически и маскируется в логах.

### Миграции и Docker

Установить CLI миграций:

```bash
go install github.com/rubenv/sql-migrate/...@latest
```

Запустить проект:

```bash
make up
make ps
```

`make up` запускает PostgreSQL, Redis и RabbitMQ, применяет миграции `development`, собирает и запускает остальные сервисы, проверяет доступность API. Эквивалент без Makefile:

```bash
docker compose up -d --wait --wait-timeout 120 postgres redis rabbitmq
"$(go env GOPATH)/bin/sql-migrate" up -env=development
docker compose up -d --build --wait --wait-timeout 120
curl -fsS --retry 30 --retry-all-errors --retry-delay 1 --max-time 5 \
  http://localhost:8001/metrics > /dev/null
```

На новой базе миграции обязательны: Docker Compose сам не создаёт таблицы `users` и `tasks`.

Если ранее использовалась очередь `image_tasks` типа Classic, см. раздел «Переход со старой очереди» перед запуском новых версий Server и Processor.

### Фронтенд

В отдельном терминале:

```bash
cd frontend
npm install
npm run dev
```

Открыть **http://localhost:5173**. Vite проксирует `/api/*` на `http://127.0.0.1:8001/*`. При локальной разработке отдельная настройка CORS не требуется. Фронтенд запускается вне Docker Compose.

### Адреса сервисов

| Сервис | Локальный адрес |
| --- | --- |
| PhotoLab | http://localhost:5173 |
| Go API | http://localhost:8001 |
| Swagger UI | http://localhost:8001/swagger/index.html |
| RabbitMQ Management | http://localhost:15672 |
| PostgreSQL | `localhost:5430` |
| Processor metrics | http://localhost:9091/metrics |
| Go API metrics | http://localhost:8001/metrics |
| Prometheus | http://localhost:9090 |
| Prometheus targets | http://localhost:9090/targets |
| Grafana | http://localhost:3000 |

Демонстрационные учётные данные: RabbitMQ — `photo` / `photo`; PostgreSQL — `photo` / `photo`, база `photo_editor`; Grafana — `admin` / `photolab_dev`. Они предназначены только для локальной разработки.

### Makefile

| Команда | Назначение |
| --- | --- |
| `make infra` | Запустить PostgreSQL, Redis, RabbitMQ |
| `make migrate` | Запустить инфраструктуру и применить миграции development |
| `make up` | Применить миграции, собрать и запустить все сервисы, дождаться API |
| `make down` | Остановить и удалить контейнеры без удаления volumes |
| `make restart` | Перезапустить проект |
| `make ps` / `make logs` | Состояние контейнеров / поток логов |
| `make build` | Собрать server и processor в `bin/` |
| `make test` / `make vet` / `make check` | Go-тесты / статический анализ / обе проверки |
| `make fmt` | Форматирование Go-кода |
| `make migrate-test` | Создать тестовую базу при необходимости, применить миграции |
| `make test-postgres` | Тесты PostgreSQL с Race Detector |
| `make test-python` | Все три набора Python-тестов (включая перезапуски сервисов) |
| `make swagger` | Перегенерировать Swagger |
| `make clean` | Удалить локальные бинарные файлы |

> Не используйте `docker compose down -v`, если хотите сохранить данные. Этот флаг удаляет volumes PostgreSQL, Redis, RabbitMQ и мониторинга.

## REST API

| Метод | Маршрут | Авторизация | Назначение |
| --- | --- | --- | --- |
| POST | `/register` | Нет | Регистрация |
| POST | `/login` | Нет | Вход и получение токена |
| POST | `/task` | Bearer | Создание задачи |
| GET | `/status/{id}` | Bearer | Статус собственной задачи |
| GET | `/result/{id}` | Bearer | Результат собственной задачи |
| POST | `/commit` | `X-Processor-Token` | Служебное завершение задачи |
| GET | `/metrics` | Нет | Метрики API |

### Регистрация и вход

```bash
curl -i -X POST http://localhost:8001/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alex","password":"123456"}'

curl -X POST http://localhost:8001/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alex","password":"123456"}'
```

Регистрация возвращает `201 Created`; повторное имя — `409 Conflict`. Успешный вход возвращает `200 OK` и JSON:

```json
{"token":"<64-символьный-hex-токен>"}
```

Пользовательский токен передаётся в заголовке `Authorization: Bearer <token>`. Сессия хранится в Redis под ключом `session:<token>` семь дней.

### Создание задачи

`POST /task` принимает PNG в Base64 и параметры фильтра:

```json
{
  "image": "<base64-encoded-png>",
  "filter": {
    "name": "Blur",
    "parameters": {"sigma": 2.5}
  }
}
```

Успешный ответ — `201 Created`:

```json
{"task_id":"<uuid>"}
```

### Статус и результат

Подставьте действительный идентификатор задачи и пользовательский токен:

```bash
curl 'http://localhost:8001/status/<uuid>' \
  -H 'Authorization: Bearer <token>'
```

(В фактической команде замените `<uuid>` целиком; не вводите угловые скобки.)

Ответ:

```json
{"status":"in_progress"}
```

Возможные статусы: `in_progress`, `ready`, `failed`. Когда задача готова, результат возвращается как PNG-байты с `Content-Type: image/png`, а не JSON:

```bash
curl -fS 'http://localhost:8001/result/<uuid>' \
  -H 'Authorization: Bearer <token>' \
  -o result.png
```

| Код | Значение |
| --- | --- |
| 200 | Статус или готовый PNG |
| 400 | Некорректный UUID или некорректный запрос |
| 401 | Нет действительной авторизации |
| 404 | Задача не существует или принадлежит другому пользователю |
| 409 | Результат недоступен (`in_progress` или `failed`) |
| 413 | Превышен лимит запроса или сообщения при создании задачи |

Чужая задача возвращает `404`, чтобы не раскрывать её существование. Старые задачи без известного владельца также недоступны через пользовательские маршруты.

### Служебный маршрут `/commit`

Processor отправляет JSON с полями `task_id`, `status` и `result` (Base64 для `ready`, пустая строка для `failed`) и отдельным заголовком:

```http
X-Processor-Token: <processor-token>
Content-Type: application/json
```

Пользовательский Bearer-токен не даёт права вызвать `/commit`.

| Ситуация | Ответ |
| --- | --- |
| Действительный токен, первый допустимый результат | `204 No Content` |
| Повтор того же конечного статуса и результата | `204 No Content` без изменения записи |
| Попытка изменить завершённую задачу | `409 Conflict` |
| Отсутствующий или неверный токен | `401 Unauthorized` |
| Некорректный UUID, статус или тело | `400 Bad Request` |
| Задача не существует | `404 Not Found` |

Переходы `in_progress → ready` и `in_progress → failed` выполняются условным атомарным `UPDATE` в PostgreSQL. Конечные статусы не перезаписываются. При конкурентных обновлениях с разными результатами только одно может успешно изменить запись.

### Фильтры

| Фильтр | Действие | Параметры |
| --- | --- | --- |
| `Negative` | Инверсия RGB с сохранением прозрачности | Нет |
| `Blur` | Размытие | `sigma`: `0 < sigma <= 20` |
| `Sharpen` | Повышение резкости | `sigma`: `0 < sigma <= 20` |
| `FlipX` | Вертикальное отражение относительно оси X | Нет |

## RabbitMQ: надёжность и повторные доставки

| Объект | Назначение |
| --- | --- |
| `image_tasks` | Основная durable quorum queue |
| `image_tasks_dlx` | Direct exchange для dead-letter сообщений |
| `image_tasks_dlq` | Отдельная durable очередь проблемных сообщений |

Server и Processor используют общую декларацию очередей (`internal/broker/queue.go`). Publisher включает подтверждения публикации и при обнаружении закрытого соединения или канала восстанавливает соединение, канал, декларации и режим confirms. Состояние подключения защищено мьютексом. Повторная отправка сообщения с неопределённым результатом публикации автоматически не производится.

Для основной очереди установлен `x-delivery-limit = 3`. Processor отклоняет некорректный JSON и постоянные ошибки commit (например, `400`, `401`, `404`, `409`) без повторной доставки. Сетевые ошибки, `408`, `429` и серверные ошибки допускают повторную доставку с паузой 3 секунды. При исчерпании лимита сообщение направляется в DLQ. Успешный commit завершается `ACK`.

Повреждённое изображение обрабатывается как неуспешная задача: Processor пытается сохранить статус `failed` через API.

Проверить очереди:

```bash
docker compose exec rabbitmq rabbitmqctl list_queues \
  name type messages_ready messages_unacknowledged
```

### Переход со старой очереди

Тип существующей Classic Queue нельзя изменить на Quorum обычным повторным объявлением. На локальном стенде прекратите отправку задач и дождитесь опустошения старой очереди. Убедитесь, что `messages_ready` и `messages_unacknowledged` равны нулю. Затем остановите приложения и удалите только пустую старую очередь:

```bash
docker compose stop server processor
docker compose exec rabbitmq rabbitmqctl delete_queue \
  image_tasks --if-empty --if-unused
make up
```

Если очередь не пуста или удаление отклонено, не удаляйте её принудительно. Данные PostgreSQL, Redis и остальные volumes при этой процедуре сохраняются.

## PostgreSQL и Redis

Миграции находятся в `migrations/`, настройки — в `dbconfig.yml`. Существуют окружения `development` (`photo_editor`) и `test` (`photo_editor_test`), доступные с хост-машины через `localhost:5430`.

- `users`: `id`, уникальный `username`, `password_hash`, `created_at`.
- `tasks`: `id`, `user_id` (внешний ключ на `users`), `status`, `result`, `created_at`. Для владельца предусмотрен индекс.
- Миграция 003 добавляет владельца задач. Для старых записей поле допускает `NULL`; новые задачи создаются с владельцем.
- `result`: PNG в Base64, поле `TEXT`.
- `schema_migrations`: учёт применённых миграций.

Проверка состояния базы:

```bash
"$(go env GOPATH)/bin/sql-migrate" status -env=development

docker compose exec postgres psql -U photo -d photo_editor \
  -c 'SELECT id, user_id, status, LENGTH(result) AS result_size, created_at FROM tasks ORDER BY created_at DESC LIMIT 5;'

docker compose exec postgres psql -U photo -d photo_editor \
  -c 'SELECT id, username FROM users;'
```

Redis хранит связь токена с UUID пользователя, не пароль и не bcrypt-хеш. TTL — `604800` секунд (7 дней). Перезапуск Go-сервера не сбрасывает действующую сессию, если Redis сохранил данные и срок сессии не истёк.

```bash
docker compose exec redis redis-cli ping
```

## Мониторинг: Prometheus + Grafana

Go API экспортирует стандартные runtime-метрики на порту `8001`, ImageProcessor — метрики на порту `9091`.

| Метрика Processor | Тип | Значение |
| --- | --- | --- |
| `processor_filter_usage_total{filter}` | Counter | Попытки применения фильтров |
| `processor_tasks_total{filter,status}` | Counter | Успешные и неуспешные обработки изображений |
| `processor_processing_duration_seconds{filter}` | Histogram | Время обработки в секундах |

Время измеряется вокруг `processImage`, а не ожидания ответа `/commit`. Повторная доставка считается отдельной попыткой обработки. Счётчики отражают попытки, а не уникальные задачи.

Prometheus опрашивает `server:8001` и `processor:9091` каждые 5 секунд. Проверка: http://localhost:9090/targets — обе цели должны быть `UP` при работающих сервисах.

Дашборд **PhotoLab — Monitoring** показывает использование фильтров, среднее время обработки и результаты обработки. Примеры PromQL:

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

Настройки находятся в `monitoring/prometheus.yml` и `monitoring/grafana/`. Grafana автоматически подключает Prometheus и загружает дашборд `dashboards/photolab.json`. Метрики конкретного фильтра появляются после первой обработки; счётчики процесса сбрасываются при перезапуске, а история Prometheus сохраняется в volume.

## Тестирование

Все команды выполняются из корня проекта.

### Go

```bash
go vet ./...
go test -v ./...
```

Go-тесты включают проверки фильтров, ограничения HTTP-тела и PNG-пикселей, PostgreSQL-репозитория и конкурентных обновлений. Без `POSTGRES_TEST_URL` тесты PostgreSQL пропускаются.

### PostgreSQL

```bash
make test-postgres
```

Makefile создаёт `photo_editor_test`, если база отсутствует, применяет миграции `test` и запускает тесты с Race Detector. Для ручного запуска (создать базу только один раз):

```bash
docker compose exec -T postgres createdb -U photo photo_editor_test
"$(go env GOPATH)/bin/sql-migrate" up -env=test
POSTGRES_TEST_URL='postgres://photo:photo@localhost:5430/photo_editor_test?sslmode=disable' \
  go test -race -v -count=1 ./...
```

Тесты репозитория проверяют сохранение, чтение, владельца, отсутствующие задачи, дубликаты, идемпотентность, конфликты и конкурентные обновления. Тестовая база должна отличаться от development; проверки репозитория требуют суффикс `_test`.

### Python

Требуются запущенные сервисы, применённые миграции и тот же токен Processor, что используется контейнерами.

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install pytest requests Pillow 'pika<2'
make test-python
```

Для прямого запуска pytest в локальном окружении:

```bash
set -a
source .env
set +a
python -m pytest -v --tb=short \
  test.py tests/course/test_course.py test_rabbitmq.py
```

| Файл | Проверки | Тестов |
| --- | --- | ---: |
| `test.py` | HTTP API, четыре фильтра, авторизация, UUID, владельцы, commit, ошибочный JSON и PNG | 22 |
| `tests/course/test_course.py` | Адаптированный набор HW3: регистрация, вход, задача, статус и PNG, 401, 404 | 6 |
| `test_rabbitmq.py` | Reconnect без перезапуска Server, DLQ для неверного сообщения, ограничение retry | 3 |
| **Всего** | | **31** |

Оригинальный PNG из набора курса находится в `tests/course/static/sigma.png`. Фикстуры создают независимых пользователей, а результат проверяется как `image/png`.

Последний предоставленный локальный прогон: **31 passed, 6 warnings**. Предупреждения относятся к устареванию Pika `BlockingConnection`; ограничение `pika<2` предотвращает установку будущей несовместимой основной версии, но не скрывает предупреждения.

> `test_rabbitmq.py` перезапускает RabbitMQ и временно останавливает Server. Выполняйте его только на локальном учебном стенде или в CI, без важных задач в обработке. Тестовые сообщения могут остаться в DLQ.

### GitHub Actions

Workflow: `.github/workflows/ci.yml`. При push и pull request CI генерирует временный токен Processor, выполняет `go vet` и сборку приложений, запускает инфраструктуру, создаёт тестовую БД, применяет миграции обоих окружений, выполняет все Go-тесты с Race Detector, запускает Docker-сервисы и три набора Python-тестов. При ошибке выводит логи контейнеров; остановка сервисов выполняется всегда.

### Swagger

```bash
go install github.com/swaggo/swag/cmd/swag@latest
make swagger
```

После изменения Swagger-аннотаций нужно перегенерировать `docs/` и пересобрать Server. Пользовательские маршруты используют Bearer-токен; для `/commit` предусмотрен отдельный заголовок `X-Processor-Token`.

## Структура проекта

```text
photo_editor/
├── cmd/
│   ├── server/                 # Go HTTP API
│   └── processor/              # Consumer и метрики
├── internal/
│   ├── broker/                 # Publisher, общая декларация очередей
│   ├── handlers/               # HTTP API, middleware, тесты лимитов
│   ├── models/                 # Модели и ошибки
│   ├── processor/              # Обработка PNG, retry, метрики и тесты
│   ├── service/                # Бизнес-логика и интерфейсы
│   └── storage/                # PostgreSQL, Redis, тесты репозитория
├── frontend/                   # React + JavaScript + Vite
├── migrations/                 # SQL-миграции
├── monitoring/                 # Prometheus и Grafana
├── docs/                       # Swagger
├── tests/course/
│   ├── test_course.py           # Адаптированные тесты курса
│   └── static/sigma.png         # PNG из HW3
├── .github/workflows/ci.yml
├── docker-compose.yml
├── rabbitmq.conf
├── dbconfig.yml
├── Dockerfile.server
├── Dockerfile.processor
├── Makefile
├── go.mod
├── test.py
└── test_rabbitmq.py
```

## Ограничения и безопасность

- `MAX_TASK_MESSAGE_SIZE = 100000000` байт: единый лимит для RabbitMQ и Server, задаваемый Docker Compose. API ограничивает размер HTTP-тела до декодирования JSON и отдельно проверяет длину сериализованного сообщения перед записью задачи и публикацией. Base64 и JSON входят в лимит; исходный PNG должен быть меньше 100 MB.
- `MaxImagePixels = 32_000_000`: Processor проверяет размеры через `png.DecodeConfig` до полного декодирования изображения. Это ограничение количества пикселей, а не гарантия фиксированного потребления памяти.
- `/commit` защищён отдельным токеном и условным обновлением в БД, но остаётся доступен на порту API. Для публичного развёртывания нужны TLS, сетевые ограничения, дополнительные лимиты служебного маршрута и безопасное управление секретами.
- Результаты хранятся в PostgreSQL как Base64 `TEXT`. Для больших объёмов разумнее использовать объектное хранилище.
- Запись в БД и публикация в RabbitMQ не объединены в одну транзакцию. При потере подтверждения результат публикации может быть неопределённым; для более строгих гарантий нужен, например, transactional outbox.
- Если commit невозможно доставить после всех повторных попыток, сообщение попадёт в DLQ, но задача в PostgreSQL может остаться `in_progress`. Автоматическое восстановление таких задач и гарантированная at-least-once доставка в DLQ не реализованы.
- Демонстрационные пароли, открытые порты API, PostgreSQL, RabbitMQ и служебные метрики не предназначены для публичного доступа. Сессия ограничена TTL: после истечения срока нужен повторный вход.