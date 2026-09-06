-include .env
export

.PHONY: db-up db-down run test tidy

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

run:
	go run ./cmd/ingestion

test:
	go test ./...

tidy:
	go mod tidy