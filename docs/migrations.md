# Миграции (goose)

## Как это работает

Файлы в `migrations/` применяются по номеру. goose хранит применённые
номера в таблице `goose_db_version` и выполняет только новые.

## Флоу

1. `make migrate-new name=create_ads` — создаётся `migrations/001_create_ads.sql`.
2. Пишешь SQL в `Up` и обратное действие в `Down` (оба в одном файле).
3. `make migrate-up` — применилась.
4. Проверка отката: `make migrate-down`, затем `make migrate-up`.

## Правила

- Применённую миграцию не правят. Нужно изменение — новая миграция.
  Пока данных нет и ты один, `001` можно переписать, сбросив базу.
- `Down` должен точно отменять `Up`. `DROP TABLE` удаляет и индексы
  таблицы, отдельный `DROP INDEX` не нужен.
- `StatementBegin`/`StatementEnd` нужны только когда внутри одного
  выражения есть `;` (функции, триггеры). Для обычного DDL не нужны.

## Каркас файла

```sql
-- +goose Up
<CREATE TABLE ...>;
<CREATE INDEX ...>;

-- +goose Down
<DROP TABLE ...>;
```

## SQL: синтаксис

```sql
CREATE TABLE <имя> (
    <колонка> <тип> [ограничения],
    <последняя_колонка> <тип> [ограничения]   -- без запятой
);

CREATE INDEX <имя_индекса> ON <таблица> (<колонка> [, <колонка> ...]);

DROP TABLE [IF EXISTS] <имя>;
```

Ограничения колонки через пробел: `PRIMARY KEY`, `NOT NULL`,
`DEFAULT <выражение>`, `CHECK (<условие>)`. Без `NOT NULL` поле nullable.

Типы: `uuid`, `varchar(<N>)`, `integer`, `bigint`,
`double precision`, `timestamptz`. Что куда — в [schema.md](schema.md).
