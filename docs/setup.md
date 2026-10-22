# Запуск и окружение

## Быстрый старт

```bash
cp .env.example .env     # один раз
make db-up               # Postgres в Docker, ждёт готовности
make migrate-up          # применить миграции
make run                 # запустить ingestion
make test                # go test ./...
```

Postgres в контейнере слушает порт **5433** на хосте (не 5432), чтобы не
конфликтовать с локально установленным Postgres. Если при `migrate-up`
видишь `role "nearwatch" does not exist`, ты подключился не к контейнеру:
проверь порт в `DATABASE_URL` (`lsof -nP -iTCP:5432 -sTCP:LISTEN`).

## Make-цели

| Цель | Что делает |
|---|---|
| `make db-up` | поднимает Postgres (`docker compose up -d --wait postgres`) |
| `make db-down` | останавливает контейнеры (данные в томе `pgdata` остаются) |
| `make migrate-up` | применяет все новые миграции |
| `make migrate-down` | откатывает одну последнюю миграцию |
| `make migrate-status` | показывает, что применено |
| `make migrate-new name=<имя>` | создаёт `migrations/NNN_<имя>.sql` с пустыми `Up`/`Down` |
| `make run` | `go run ./cmd/ingestion` |
| `make test` | `go test ./...` |
| `make tidy` | `go mod tidy` |

Цели с миграциями читают `DATABASE_URL` из `.env`. goose запускается
через `go run ...@v3.26.0`, отдельно ставить не нужно.

## Конфиг (env)

Читается в `internal/ingestion/config/config.go`.

| Переменная | Обязательна | По умолчанию | Примечание |
|---|---|---|---|
| `DATABASE_URL` | да | — | без неё сервис не стартует |
| `HTTP_ADDR` | нет | `:8080` | |
| `LOG_LEVEL` | нет | `info` | `debug`/`info`/`warn`/`error`; неизвестное значение молча = `info` |
| `SHUTDOWN_TIMEOUT` | нет | `10s` | кривое значение — ошибка старта |

`LOG_LEVEL` — порог: `warn` скрывает `info` и `debug`.
