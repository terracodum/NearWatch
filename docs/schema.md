# Схема БД

Источник: [README, модель данных](../README.md#модель-данных) и
[схема БД](../README.md#схема-бд). Реальная схема определяется
миграциями в `migrations/`.

## Таблица `ads` (Этап 1)

| Поле | Тип в Postgres | Заметки |
|---|---|---|
| id | `uuid` | PK, дефолт `gen_random_uuid()` |
| title | `varchar(200)` | `NOT NULL` |
| description | `varchar(4000)` | `NOT NULL` |
| price | `bigint` | `NOT NULL`, копейки, не отрицательная |
| category_id | `integer` | `NOT NULL` |
| city_id | `integer` | `NOT NULL` |
| lat, lon | `double precision` | nullable |
| created_at | `timestamptz` | `NOT NULL`, дефолт `now()` |

Индексы:

- `ads(created_at)`
- `ads(category_id, city_id)`

## Позже (Этапы 3–4)

Таблицы `filters`, `matches`, `users` и подписки пока не создаются:
модель `filters`/`matches` может измениться по
[ADR 0001](decisions/0001-shared-filters-and-index.md).
