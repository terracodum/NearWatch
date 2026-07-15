# NearWatch

NearWatch — сервис подписки на поиск: пользователь сохраняет фильтр, система следит за появлением новых объявлений и мгновенно уведомляет о совпадениях через WebSocket.

Упрощённая реализация внутренней задачи маркетплейса («уведомить о новом объявлении по сохранённому поиску») на микросервисной архитектуре.

---

## Архитектура

```text
feed-generator ──gRPC client-stream──> ingestion ──> PostgreSQL
                                           │
                                           ▼
                                  Redis: ad.created
                                           │
                                           ▼
                          matcher (worker pool + кэш фильтров)
                                           │
                                           ▼
                                  Redis: match.found
                                           │
                                           ▼
                                       gateway ──WebSocket──> клиент
```

Принципы:

- Каждый сервис владеет своими таблицами и не пишет в чужие
- Связь между сервисами: gRPC (синхронно) и Redis Pub/Sub (асинхронно, fire-and-forget)
- Единственная точка входа для клиента — gateway

Владение данными:

| Таблица | Пишет | Читает |
|---|---|---|
| `ads` | ingestion | matcher, gateway |
| `filters` | gateway | matcher |
| `matches` | matcher | gateway |
| `users` | gateway | — |

---

## Сервисы

### feed-generator

Эмитит поток фейковых объявлений с настраиваемым RPS. Не является частью продукта — это управляемый источник нагрузки.

- Генерирует объявления по gRPC client-stream в ingestion
- RPS настраивается через конфиг (для нагрузочных прогонов)
- Позволяет воспроизводимо замерить пропускную способность системы

### ingestion

Приём и сохранение объявлений.

- gRPC-сервер, принимает client-stream объявлений
- Валидация, запись в PostgreSQL в транзакции
- Публикация события `ad.created` в Redis
- Не знает о существовании фильтров и матчей

### matcher

Сопоставление объявлений с фильтрами. Основная вычислительная нагрузка.

- Подписан на `ad.created` и `filter.changed`
- Держит активные фильтры в памяти (кэш под RWMutex, наполняется из `filters` при старте, инвалидация по `filter.changed`)
- Worker pool: параллельная проверка объявления против всех фильтров
- Пишет совпадение в `matches`, публикует `match.found`
- В `ads` и `filters` только читает

### gateway

Точка входа для клиента.

- REST API: CRUD фильтров (пишет в `filters`), регистрация и логин (пишет в `users`)
- При любом изменении фильтра публикует `filter.changed` — иначе кэш matcher протухнет
- WebSocket-хаб: клиент подписывается на свои матчи
- Подписан на `match.found`, роутит матч нужному соединению
- Broadcast не блокируется на медленном клиенте (буфер + drop policy)
- Написан на чистом `net/http` без фреймворка

---

## Модель данных

### Ad (объявление)

| Поле | Тип | Описание |
|---|---|---|
| id | UUID | Первичный ключ |
| title | string | Заголовок, до 200 символов |
| description | string | Описание, до 4000 символов |
| price | int64 | Цена в копейках (не float — деньги) |
| category_id | int | Категория |
| city_id | int | Город |
| lat, lon | float64 | Координаты (nullable) |
| created_at | timestamptz | Время создания |

### Filter (сохранённый поиск)

| Поле | Тип | Описание |
|---|---|---|
| id | UUID | Первичный ключ |
| user_id | UUID | Владелец |
| query | string | Подстрока для поиска в title (nullable) |
| category_id | int | Категория (nullable) |
| city_id | int | Город (nullable) |
| price_min | int64 | Нижняя граница цены (nullable) |
| price_max | int64 | Верхняя граница цены (nullable) |
| is_active | bool | Отключённый фильтр не участвует в матчинге |
| created_at | timestamptz | Время создания |

Правило: `nullable` поле означает «не фильтруем по этому признаку». Фильтр без единого заданного поля невалиден.

### Match (совпадение)

| Поле | Тип | Описание |
|---|---|---|
| id | UUID | Первичный ключ |
| filter_id | UUID | Какой фильтр сработал |
| ad_id | UUID | Какое объявление подошло |
| user_id | UUID | Кому доставлять |
| created_at | timestamptz | Время совпадения |

Уникальность по паре `(filter_id, ad_id)` — одно объявление не матчится дважды на один фильтр.

### User

| Поле | Тип | Описание |
|---|---|---|
| id | UUID | Первичный ключ |
| email | string | Уникальный |
| password_hash | string | bcrypt |
| created_at | timestamptz | Время регистрации |

---

## Схема БД

```text
users
  │ 1
  │
  │ N
filters ──1───< matches >───N── ads
                    N              1
```

- `filters.user_id` → `users.id` (CASCADE при удалении пользователя)
- `matches.filter_id` → `filters.id` (CASCADE — удалил фильтр, история матчей по нему не нужна)
- `matches.ad_id` → `ads.id` (RESTRICT — объявления не удаляем)
- `matches.user_id` → `users.id` (денормализация: чтобы gateway не джойнил на каждый пуш)

Индексы:

- `ads(created_at)` — выборка свежих
- `ads(category_id, city_id)` — типовой запрос
- `filters(is_active)` — загрузка активных фильтров в кэш matcher
- `matches(user_id, created_at)` — история матчей пользователя
- `matches(filter_id, ad_id)` UNIQUE — защита от дублей на уровне БД

---

## Контракты

### gRPC (proto)

```protobuf
service IngestionService {
  rpc StreamAds(stream Ad) returns (StreamAdsResponse);
}

message Ad {
  string title = 1;
  string description = 2;
  int64 price = 3;
  int32 category_id = 4;
  int32 city_id = 5;
  optional double lat = 6;
  optional double lon = 7;
}

message StreamAdsResponse {
  int64 accepted = 1;
  int64 rejected = 2;
}
```

### Redis Pub/Sub

Канал `ad.created`:

```json
{
  "id": "uuid",
  "title": "iPhone 13 Pro",
  "price": 5500000,
  "category_id": 12,
  "city_id": 55,
  "created_at": "2026-07-16T10:00:00Z"
}
```

Канал `match.found`:

```json
{
  "match_id": "uuid",
  "user_id": "uuid",
  "filter_id": "uuid",
  "ad_id": "uuid",
  "ad_title": "iPhone 13 Pro",
  "ad_price": 5500000
}
```

Канал `filter.changed` (инвалидация кэша matcher):

```json
{
  "filter_id": "uuid",
  "action": "created | updated | deleted"
}
```

**Важно:** Redis Pub/Sub не гарантирует доставку. Если matcher был отключён — события потеряны. Это осознанный trade-off для текущих требований. Переход на Redis Streams / Kafka — при требовании гарантированной доставки.

### REST API (gateway)

```text
POST   /api/v1/auth/register     — регистрация
POST   /api/v1/auth/login        — получить JWT

GET    /api/v1/filters           — список своих фильтров
POST   /api/v1/filters           — создать фильтр
PATCH  /api/v1/filters/:id       — изменить фильтр
DELETE /api/v1/filters/:id       — удалить фильтр

GET    /api/v1/matches           — история матчей (пагинация)

GET    /api/v1/ws                — WebSocket upgrade (JWT в query)
```

### WebSocket протокол

Сервер → клиент:

```json
{
  "type": "match",
  "payload": {
    "match_id": "uuid",
    "filter_id": "uuid",
    "ad_title": "iPhone 13 Pro",
    "ad_price": 5500000
  }
}
```

Клиент → сервер: только `ping`. Подписка неявная — клиент получает матчи своего `user_id` из JWT.

---

## Структура проекта

```text
nearwatch/
├── proto/                      # общие контракты
│   ├── ingestion/v1/
│   │   └── ingestion.proto
│   └── gen/                    # сгенерированный код (make proto)
│
├── services/
│   ├── ingestion/
│   │   ├── cmd/
│   │   ├── internal/
│   │   │   ├── domain/         # сущности, бизнес-правила
│   │   │   ├── repository/     # database/sql
│   │   │   ├── grpc/           # gRPC-хендлеры
│   │   │   └── publisher/      # Redis publish
│   │   └── migrations/
│   │
│   ├── matcher/
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── domain/         # логика матчинга
│   │       ├── cache/          # кэш фильтров под RWMutex
│   │       ├── pool/           # worker pool
│   │       └── subscriber/     # Redis subscribe
│   │
│   ├── gateway/
│   │   ├── cmd/
│   │   └── internal/
│   │       ├── handler/        # net/http
│   │       ├── auth/           # JWT
│   │       ├── hub/            # WebSocket-хаб
│   │       └── subscriber/
│   │
│   └── feed-generator/
│       ├── cmd/
│       └── internal/
│
├── docs/
│   ├── architecture.md
│   └── decisions/              # ADR: почему Pub/Sub, почему matcher отдельно
│
├── docker-compose.yml
├── .env.example
├── go.work                     # связка модулей монорепы
├── Makefile
└── README.md
```

---

## Технологии

- **Go** — все сервисы, монорепа через `go.work`
- **PostgreSQL** — хранение (`database/sql`, без ORM)
- **Redis** — Pub/Sub между сервисами
- **gRPC** — feed-generator → ingestion
- **WebSocket** — gateway → клиент
- **goose** — миграции
- **slog** — структурированное логирование
- **Prometheus** — метрики
- **Docker Compose** — локальный запуск

---

## Запуск

```bash
git clone https://github.com/terracodum/nearwatch.git
cd nearwatch
cp .env.example .env
make proto              # генерация кода из .proto
docker-compose up --build
```

- Gateway: http://localhost:8080
- Prometheus: http://localhost:9090

Нагрузочный прогон:

```bash
make load RPS=1000
```

---

## План разработки

Каждый этап — работающая система, а не полуфабрикат.

### Этап 1 — скелет

`ingestion` с HTTP-эндпоинтом `POST /ads`, запись в Postgres. Миграции, конфиг через env, slog, graceful shutdown.

**Демо:** curl создаёт объявление, оно в базе.

### Этап 2 — контракты

Proto-контракт в `proto/`, ingestion переходит на gRPC client-stream, появляется feed-generator.

**Демо:** генератор льёт 100 объявлений/сек, они падают в базу.

### Этап 3 — конкурентность

Redis Pub/Sub, matcher с worker pool, кэш фильтров под RWMutex, backpressure, context с таймаутом.

**Демо:** матчер обрабатывает поток, в логах видны совпадения.

### Этап 4 — realtime

Gateway на `net/http`, WebSocket-хаб, JWT, broadcast без блокировки.

**Демо:** объявление появляется в браузере через долю секунды после генерации.

### Этап 5 — цифры

Prometheus, pprof, бенчмарки, нагрузочный прогон. Поиск узкого места, оптимизация, повторный замер.

**Демо:** «выдерживает N объявлений/сек, p99 = X мс».

Тесты (table-driven + моки через интерфейсы) пишутся на каждом этапе, не в конце.

---

## Архитектурные решения

| Решение | Почему | Trade-off |
|---|---|---|
| matcher отдельным сервисом | Разный профиль нагрузки: ingestion упирается в I/O, matcher в CPU | Дополнительная сетевая задержка |
| Redis Pub/Sub вместо прямого вызова | ingestion не ждёт matcher и не падает вместе с ним | Нет гарантии доставки |
| Кэш фильтров в памяти matcher | На каждое объявление нужен полный перебор фильтров — запрос в БД убил бы latency | Инвалидация через `filter.changed`, рассинхрон при рестарте |
| Цена в int64 (копейки) | Деньги нельзя во float | Нужна конвертация на границе API |
| `net/http` вместо Gin в gateway | Понимать, что фреймворк делает под капотом | Больше кода на роутинг |
| PostgreSQL вместо SQLite | Реальный пул соединений, прод-подобное поведение | Тяжелее локальный запуск |
| Монорепа без submodules | Один разработчик, один контракт — submodules дают только оверхед на синхронизацию | При росте до нескольких команд контракт пришлось бы выносить |
| `matches.user_id` денормализован | gateway не джойнит на каждый пуш — user_id уже в событии | Дублирование данных, рассинхрон при смене владельца фильтра (не бывает) |

## Известные ограничения

Не баги, а осознанные границы текущей версии. Каждое — заготовленный ответ на «а что если».

**matcher масштабируется только вертикально.** Кэш фильтров лежит в памяти процесса, поэтому два инстанса обработают одно объявление дважды. От дублей в БД спасает `UNIQUE (filter_id, ad_id)`, но `match.found` уйдёт дважды и клиент получит два пуша. Горизонтальное масштабирование потребует шардирования фильтров по инстансам (consistent hashing) или вынесения кэша в общий Redis.

**Pub/Sub теряет события.** Матчер лежал минуту — объявления за эту минуту не проматчены, и никогда не будут. Восстановление потребует Redis Streams с consumer groups либо периодического reconcile-прохода по `ads` с `created_at` больше последнего обработанного.

**Кэш фильтров прогревается при старте.** Между запуском процесса и загрузкой фильтров из БД есть окно, где матчер видит пустой набор. Решается блокировкой подписки до завершения прогрева.

**Матчинг по подстроке — линейный.** `query` проверяется через `strings.Contains` по всем фильтрам. При десятках тысяч фильтров это упрётся в CPU — тогда нужен инвертированный индекс или префиксное дерево. До этого этап 5 покажет, где реальный предел.
