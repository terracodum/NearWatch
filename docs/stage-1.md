# Этап 1: скелет

Цель: `POST /ads` на чистом `net/http` кладёт объявление в Postgres.

Не делать на этом этапе: gRPC, Redis, матчер, JWT, WebSocket, метрики.
Идеи, которые тянет сделать раньше, — в «морозилку идей» в Notion.

## Чеклист

- [x] Каркас и структура папок
- [x] Конфиг из env + тесты
- [x] docker-compose с Postgres, цели миграций
- [x] Миграции `001_create_ads` и `002_add_ads_indexes` ([как](migrations.md), [схема](schema.md))
- [ ] slog в JSON-режиме, передаётся через конструкторы
- [ ] `domain.Ad` + `Validate()` + table-driven тест
- [ ] `AdRepository` + реализация (`database/sql`, пул настроен осознанно)
- [ ] `POST /ads` (201/400/500), `GET /healthz`, таймауты сервера
- [ ] Graceful shutdown (`signal.NotifyContext`, `Shutdown` 10 с, пул закрывать после сервера)
- [ ] Тест хендлера с ручным моком репозитория

## Готово, когда

```bash
docker compose up --build
curl -X POST localhost:8080/ads -H 'Content-Type: application/json' -d '{
  "title": "iPhone 13 Pro",
  "description": "Состояние отличное",
  "price": 5500000,
  "category_id": 12,
  "city_id": 55
}'
# -> 201 Created + id

curl localhost:8080/healthz   # -> 200 OK
go test ./...                 # -> всё зелёное
```

Ctrl+C: в логах аккуратное завершение, а не обрыв.

Подробный чеклист со ссылками на что читать — в Notion, страница
«Этап 1 — скелет» внутри проекта NearWatch.
