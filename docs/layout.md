# Структура репо

```text
cmd/ingestion/               точка входа (main.go)
internal/ingestion/
  config/                    env-конфиг + тесты
  domain/                    Ad, Validate(), интерфейс AdRepository
  repository/                database/sql + pgx
  httpapi/                   net/http хендлеры
migrations/                  goose, SQL
docs/                        рабочая документация и ADR
docker-compose.yml           Postgres (ingestion добавится с Dockerfile)
```

## Расхождения README и кода

- README описывает `go.work` и `services/ingestion/`; в коде один
  корневой `go.mod` и `internal/ingestion/` (решено сознательно:
  `go.work` ничего не давал). README стоит поправить.
- Модель `filters`/`matches` в README устареет, если принять
  [ADR 0001](decisions/0001-shared-filters-and-index.md).
