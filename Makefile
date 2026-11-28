.DEFAULT_GOAL := up

.PHONY: up down restart logs ps build test vet fmt check clean \
        infra migrate migrate-test test-postgres test-python swagger wait-api

PYTHON ?= python3
SQL_MIGRATE ?= $(shell go env GOPATH)/bin/sql-migrate
SWAG ?= $(shell go env GOPATH)/bin/swag
POSTGRES_TEST_URL ?= postgres://photo:photo@localhost:5430/photo_editor_test?sslmode=disable

infra:
	docker compose up -d --wait --wait-timeout 120 postgres redis rabbitmq

migrate: infra
	"$(SQL_MIGRATE)" up -env=development

up: migrate
	docker compose up -d --build --wait --wait-timeout 120
	$(MAKE) wait-api

wait-api:
	curl --fail --silent --show-error --retry 30 --retry-all-errors \
		--retry-delay 1 --max-time 5 http://localhost:8001/metrics > /dev/null

down:
	docker compose down

restart:
	docker compose down
	$(MAKE) up

logs:
	docker compose logs -f

ps:
	docker compose ps

build:
	mkdir -p bin
	go build -o bin/server ./cmd/server
	go build -o bin/processor ./cmd/processor

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

check: vet test

migrate-test: infra
	@docker compose exec -T postgres psql -U photo -d postgres -tAc \
		"SELECT 1 FROM pg_database WHERE datname = 'photo_editor_test'" | grep -qx 1 || \
		docker compose exec -T postgres createdb -U photo photo_editor_test
	"$(SQL_MIGRATE)" up -env=test

test-postgres: migrate-test
	POSTGRES_TEST_URL="$(POSTGRES_TEST_URL)" go test -race -v -count=1 ./internal/storage

test-python: wait-api
	@set -e; \
	if [ -z "$$PROCESSOR_TOKEN" ] && [ -f .env ]; then \
		set -a; . ./.env; set +a; \
	fi; \
	: "$${PROCESSOR_TOKEN:?PROCESSOR_TOKEN must be set}"; \
	$(PYTHON) -m pytest -v --tb=short test.py tests/course/test_course.py test_rabbitmq.py

swagger:
	"$(SWAG)" init -g cmd/server/main.go

clean:
	rm -rf bin