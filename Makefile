-include .env
export

GOOSE = GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" GOOSE_MIGRATION_DIR=migrations \
	go run github.com/pressly/goose/v3/cmd/goose@v3.26.0

.PHONY: db-up db-down check-db-url migrate-up migrate-down migrate-status migrate-new run test tidy

check-db-url:
	@test -n "$(DATABASE_URL)" || { echo "DATABASE_URL не задан. Выполни: cp .env.example .env"; exit 1; }

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

migrate-up: check-db-url
	$(GOOSE) up

migrate-down: check-db-url
	$(GOOSE) down

migrate-status: check-db-url
	$(GOOSE) status

# make migrate-new name=create_ads -> migrations/001_create_ads.sql (чистый шаблон)
migrate-new:
	@test -n "$(name)" || { echo "usage: make migrate-new name=create_ads"; exit 1; }
	@n=$$(printf '%03d' $$(( $$(ls migrations/*.sql 2>/dev/null | wc -l) + 1 ))); \
	f=migrations/$${n}_$(name).sql; \
	printf -- '-- +goose Up\n\n\n-- +goose Down\n\n' > $$f; \
	echo "created $$f"

run:
	go run ./cmd/ingestion

test:
	go test ./...

tidy:
	go mod tidy