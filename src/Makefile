DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/tms_db?sslmode=disable
MIGRATIONS_DIR := db/migrations

.PHONY: docker-up docker-down migrate-up migrate-down migrate-create sqlc-generate build

docker-up:
	cd .. && docker compose up -d

docker-down:
	cd .. && docker compose down

migrate-up:
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS_DIR) up

migrate-down:
	migrate -database "$(DATABASE_URL)" -path $(MIGRATIONS_DIR) down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq -digits 6 $$name

sqlc-generate:
	sqlc generate

build:
	go build ./...
