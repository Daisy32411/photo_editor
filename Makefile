.PHONY: up down restart logs ps build test vet fmt check clean

# СОбрать docker образы и заупстить все сервисы
up:
	docker compose up -d --build

# Остановить и удалить контейнеры проекта 
down:
	docker compose down

# Перезапустить весь проект
restart:
	docker compose down
	docker compose up -d --build

# Смотреть логи всех контейнеров
logs:
	docker compose logs -f

# Показать состояние контейнеров
ps:
	docker compose ps

# Скомпилировать оба Go-приложения
build:
	mkdir -p bin
	go build -o bin/server ./cmd/server
	go build -o bin/processor ./cmd/processor


# Запустить Go-тесты
test:
	go test ./...

# Проверить код с помощью go vet
vet:
	go vet ./...

# Отформатировать Go-код
fmt:
	go fmt ./...

# Выполнить основные проверки
check:
	go vet ./...
	go test ./...

# Удалить локальные скомпилированные программы
clean:
	rm -rf bin