# Photo Editor

**Photo Editor** — сервис асинхронной обработки PNG-изображений на Go. HTTP-сервер принимает задачи, RabbitMQ доставляет их отдельному процессору изображений, а клиент может проверить статус и скачать готовый PNG.

## Возможности

- Регистрация пользователей и вход с токеном сессии.
- Асинхронное выполнение задач через RabbitMQ.
- Четыре фильтра: `Negative`, `Blur`, `Sharpen`, `FlipX`.
- Статусы задач: `in_progress`, `ready`, `failed`.
- Скачивание обработанного изображения в формате PNG.
- Подтверждение публикаций RabbitMQ (Publisher Confirms) и ручное подтверждение обработки сообщений (`Ack`/`Nack`).
- Swagger UI, Docker Compose, Makefile, unit-тесты Go и интеграционные тесты Python.
- GitHub Actions CI.

## Архитектура

```text
                  Клиент
                     |
       POST /register, POST /login
       POST /task, GET /status/{id}
       GET /result/{id}
                     |
                     v
           +-------------------+
           |   HTTP-сервер Go  |
           |       :8001       |
           +-------------------+
             |              ^
        Publish           POST /commit
             |              |
             v              |
           +-------------------+
           |     RabbitMQ      |
           |  очередь image_tasks
           +-------------------+
                     |
                  Consume
                     |
                     v
           +-------------------+
           |   ImageProcessor  |
           |   отдельный Go    |
           |      процесс      |
           +-------------------+
```

Сервер и ImageProcessor — **два отдельных исполняемых приложения** в одном Go-модуле. В очереди передаются ID задачи, PNG в Base64 и настройки фильтра. Обработчик возвращает результат серверу через `POST /commit`.

**Важно:** пользователи, сессии, статусы и результаты задач хранятся **в оперативной памяти HTTP-сервера**. При его перезапуске эти данные пропадают. RabbitMQ не заменяет постоянное хранилище задач.

## Технологии

- Go
- RabbitMQ и [`amqp091-go`](https://github.com/rabbitmq/amqp091-go)
- [`imaging`](https://github.com/disintegration/imaging) для Blur и Sharpen
- Docker / Docker Compose
- Swagger / Swaggo
- Python, pytest, requests, Pillow
- GitHub Actions

## Быстрый запуск

### Требования

- Docker
- Docker Compose (`docker compose`)

Go и Python нужны для локальной разработки и запуска тестов вне Docker.

```bash
git clone https://github.com/Daisy32411/photo_editor.git
cd photo_editor

docker compose up -d --build
```

Или через Makefile:

```bash
make up
```

Проверить сервисы и логи:

```bash
docker compose ps
docker compose logs -f
```

Остановить проект:

```bash
make down
```

### Адреса

| Сервис | Адрес |
| --- | --- |
| HTTP API | `http://localhost:8001` |
| Swagger UI | `http://localhost:8001/swagger/index.html` |
| RabbitMQ Management | `http://localhost:15672` |

Данные для входа в RabbitMQ Management при локальной разработке: `photo` / `photo`.

> Учебные логин и пароль не подходят для публичного развёртывания.

## API

### Регистрация и вход

| Метод | Путь | Назначение |
| --- | --- | --- |
| `POST` | `/register` | Регистрация (`username`, `password`) |
| `POST` | `/login` | Получение токена сессии |

**Регистрация:**

```bash
curl -i -X POST http://localhost:8001/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alex","password":"123456"}'
```

При успешной регистрации сервер возвращает `201 Created` без тела ответа. При повторной регистрации того же имени — `409 Conflict`.

**Вход и сохранение токена:**

```bash
TOKEN=$(curl -fsS -X POST http://localhost:8001/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alex","password":"123456"}' \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')
```

Следующие пользовательские эндпоинты требуют заголовка:

```text
Authorization: Bearer <token>
```

### Обработка изображения

| Метод | Путь | Назначение |
| --- | --- | --- |
| `POST` | `/task` | Создать задачу на обработку PNG |
| `GET` | `/status/{id}` | Узнать статус задачи |
| `GET` | `/result/{id}` | Получить результат в формате `image/png` |
| `POST` | `/commit` | Служебный приём результатов от ImageProcessor |

**Тело `POST /task`:**

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

**Пример подготовки запроса из `test.png` без сторонних Python-библиотек:**

```bash
python3 - <<'PY'
import base64
import json
from pathlib import Path

image = Path('test.png').read_bytes()
payload = {
    'image': base64.b64encode(image).decode('ascii'),
    'filter': {'name': 'Negative'},
}
Path('task.json').write_text(json.dumps(payload), encoding='utf-8')
PY
```

Файл `test.png` нужно положить в текущую папку. Отправить задачу:

```bash
TASK_ID=$(curl -fsS -X POST http://localhost:8001/task \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  --data-binary @task.json \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["task_id"])')

echo "$TASK_ID"
```

Успешный ответ — `201 Created` и JSON с `task_id`.

**Проверить статус:**

```bash
curl -s "http://localhost:8001/status/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN"
```

Возможные значения:

```json
{"status":"in_progress"}
```

```json
{"status":"ready"}
```

```json
{"status":"failed"}
```

**Скачать результат** после получения `ready`:

```bash
curl -fS "http://localhost:8001/result/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -o result.png
```

Если задача ещё не готова или завершилась ошибкой, `/result/{id}` возвращает `409 Conflict`.

### Поддерживаемые фильтры

| Имя | Действие | Параметры |
| --- | --- | --- |
| `Negative` | Инверсия RGB, сохранение альфа-канала | Нет |
| `Blur` | Размытие изображения | `sigma`: число, `0 < sigma <= 20` |
| `Sharpen` | Повышение резкости | `sigma`: число, `0 < sigma <= 20` |
| `FlipX` | Отражение относительно оси X (сверху вниз) | Нет |

Примеры объекта `filter`:

```json
{"name":"Negative"}
```

```json
{"name":"Blur","parameters":{"sigma":2.5}}
```

```json
{"name":"Sharpen","parameters":{"sigma":1.5}}
```

```json
{"name":"FlipX"}
```

### Служебный эндпоинт `/commit`

ImageProcessor отправляет туда статус и результат задачи:

```json
{
  "task_id": "<uuid>",
  "status": "ready",
  "result": "<base64-encoded-png>"
}
```

При неудачной обработке используется `"status":"failed"` и пустая строка `result`. При успешном сохранении сервер отвечает `204 No Content`.

> В текущей учебной реализации `/commit` не имеет отдельной авторизации. Не публикуйте этот API в интернете без защиты служебного маршрута.

## Размеры изображений и RabbitMQ

Лимит сообщения RabbitMQ увеличен до **100 МиБ** вместо прежних 50 МиБ. Для такого лимита в `rabbitmq.conf` используется:

```ini
max_message_size = 104857600
```

Это ограничение **размера AMQP-сообщения**, а не размера исходного PNG. Передача PNG через Base64 увеличивает его размер примерно на треть.

У HTTP-сервера есть **отдельная проверка** `MaxTaskMessageSize` в `internal/service/task.go`. Её значение должно быть согласовано с лимитом RabbitMQ (меньше максимального размера сообщения брокера); иначе сервер может возвращать `413` ещё до отправки задачи. Фактический допустимый размер запроса определяется этой проверкой.

Проверить конфигурацию запущенного RabbitMQ:

```bash
docker compose exec rabbitmq rabbitmq-diagnostics environment | grep max_message_size
```

## Команды Makefile

| Команда | Назначение |
| --- | --- |
| `make up` | Собрать Docker-образы и запустить сервисы |
| `make down` | Остановить сервисы |
| `make restart` | Перезапустить сервисы |
| `make ps` | Показать состояние контейнеров |
| `make logs` | Показать логи |
| `make build` | Собрать Go-сервер и ImageProcessor в `bin/` |
| `make test` | Запустить Go-тесты |
| `make vet` | Запустить `go vet` |
| `make fmt` | Отформатировать Go-код |
| `make check` | Выполнить `go vet` и Go-тесты |
| `make clean` | Удалить локальные бинарные файлы из `bin/` |

## Тестирование

### Unit-тесты Go

Не требуют Docker или запущенного RabbitMQ:

```bash
make check
```

Для подробного вывода тестов фильтров:

```bash
go test -v ./internal/processor
```

Тестируются фильтры `Negative`, `Blur`, `Sharpen`, `FlipX`, разбор `sigma` и неизвестное имя фильтра.

### Интеграционные тесты Python

Проверяют полный путь: HTTP → RabbitMQ → ImageProcessor → HTTP и получение PNG.

Для них необходимы **запущенные сервисы**:

```bash
make up
python3 -m venv .venv
source .venv/bin/activate
python -m pip install pytest requests Pillow
python -m pytest -v test.py
```

Интеграционные тесты запускаются из файла `test.py`.

### GitHub Actions CI

При отправке изменений GitHub Actions выполняет проверки Go, сборку приложений и интеграционные Python-тесты с Docker Compose.

## Swagger

Документация доступна по адресу:

```text
http://localhost:8001/swagger/index.html
```

После изменения Swagger-аннотаций в Go-коде пересоздайте документацию:

```bash
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
```

Затем пересоберите сервер:

```bash
make up
```

## Известные ограничения

- Данные задач и сессий не сохраняются между перезапусками HTTP-сервера.
- `/commit` пока не защищён отдельным механизмом авторизации.
- Повторные попытки обработки временных ошибок пока не имеют максимального числа повторов или отдельной Dead Letter Queue.
- Нет автоматического восстановления AMQP-соединений после обрыва; может потребоваться перезапуск Go-сервисов.
- Текущая реализация ориентирована на учебный и локальный запуск, а не на публичную production-среду.
