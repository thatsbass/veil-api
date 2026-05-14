.PHONY: run build test lint migrate-up migrate-down sqlc docker-up docker-down

BINARY_NAME=veil
MIGRATE=migrate -path migrations -database "$(DATABASE_URL)"

run:
	go run cmd/server/main.go

build:
	go build -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd/server

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

sqlc:
	sqlc generate -f sqlc/sqlc.yaml

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-reset:
	docker-compose down -v

tidy:
	go mod tidy
