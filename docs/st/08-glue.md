# Модуль 08 — Glue (HTTP server composition + e2e)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 08 (Glue): собрать все модули в один HTTP сервис, настроить middleware chain и обеспечить минимальный e2e тест.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/08-glue.md.
- Контракты (OpenAPI) и миграции БД задаёт Модуль 00. Считай их “замороженными”.
- Auth (01) и Metrics (07) должны быть подключены middleware’ами.
- Решения: TokenStore — production в internal/auth/sqlstore.go; embedding stub в основном коде (internal/embedding или search); e2e создаёт токены напрямую в БД; цепочка Metrics→Auth→EnrichPrincipal, затем по маршрутам RequireStaff/RequireAppOrStaff; CaseCreator-адаптер в Glue; явная проверка pgvector после миграций; e2e в integration/e2e. См. секцию «Уточнения».

Ограничения по изменениям (важно):
- Не меняй contracts/openapi.yaml и db/migrations/* (если только не чинится очевидная ошибка в контрактах, и то лучше через владельца модуля 00).
- Не добавляй бизнес-логику в Glue: только wiring/инициализация.

Куда класть код (если структура проекта ещё не задана):
- cmd/api/* (main, запуск сервера)
- internal/http/* (router, server)
- internal/config/* (конфиг)
- docker-compose.yml (опционально, для локальной интеграции)
Если структура проекта уже есть — следуй ей.

Что нужно сделать (порядок):
1) Инициализируй подключение к Postgres, убедись что миграции накатываются.
2) Собери зависимости модулей и подними HTTP сервер с маршрутами согласно OpenAPI.
3) Подключи middleware chain: Metrics -> Auth -> Handlers (метрики первыми, чтобы писать все запросы, включая 401/403).
4) Сконфигурируй embedding provider (можно stub/mock для тестов) и Template renderer.
5) Реализуй минимальный e2e тест (см. README):
   - создать app/staff токены (через прямые вставки в БД или через admin API, если есть)
   - пройти цикл кейса до approved и поиск
   - создать/обработать тикет и convert-to-case
6) Убедись, что `go test ./...` проходит (включая integration/e2e).

Definition of Done:
- Сервис стартует и отвечает по контракту.
- Минимальный e2e тест стабильно проходит на чистой БД.

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- как запускать сервис локально;
- как прогонять e2e/integration тесты.
```

## Цель
Собрать все модули в единый сервис:
- роутинг
- цепочки middleware (metrics → auth → handler; см. 07-metrics: метрики первыми)
- dependency injection (DB pool, repos, renderer, embedding provider)
- минимальный e2e прогон

## Границы модуля
В Glue **не допускается** бизнес-логика (никаких SQL/правил статусов/поиска).
Только wiring и конфигурация.

## Уточнения (решения по открытым вопросам)

1. **TokenStore для продакшена** — Реализацию **вынести в отдельный файл без тега**: `internal/auth/sqlstore.go`. Использовать её и в интеграционных тестах auth (импорт из того же пакета), и в Glue. Интерфейс остаётся в auth; production-реализация живёт в пакете auth, а не в Glue.

2. **EmbeddingProvider для e2e и локального запуска** — Заглушка (фиксированный вектор или нули) живёт **в основном коде**: например `internal/embedding/stub.go` (или `internal/search/embedding_stub.go`). Использовать в e2e и при «обычном» локальном запуске без реального провайдера (режим dev/demo). Без тегов: участвует в обычном `go build`.

3. **Создание app/staff токенов в e2e** — Отдельного admin API в OpenAPI нет. **E2e создаёт записи в `apps` и `auth_tokens` напрямую**: через хелпер с тем же DSN, миграциями и репозиторием/сырым SQL (вставка app, затем auth_tokens с посчитанным token_hash). Без нового эндпоинта.

4. **Цепочка middleware (EnrichPrincipal и доступ)** — Верно: **одна общая цепочка** до точки ветвления: `Metrics(writer) → Authenticate(store, secret) → EnrichPrincipal → [далее по маршруту]`. Затем **по группам маршрутов**: для staff-only после EnrichPrincipal вешается `RequireStaff`, затем handler; для search (app+staff) — `RequireAppOrStaff`, затем handler. EnrichPrincipal один раз на всё приложение; RequireStaff/RequireAppOrStaff — по роутам.

5. **CaseCreator: адаптер cases → tickets** — Адаптер, реализующий `tickets.CaseCreator` и вызывающий `cases.Service.Create` (маппинг CreateDraftFromTicketRequest → CaseCreate), класть **в Glue**: например `internal/http` или отдельный пакет `internal/glue`. Не в пакеты cases и tickets.

6. **Проверка pgvector при старте** — **Явная проверка после миграций**: например `SELECT 1 FROM pg_extension WHERE extname = 'vector'`; при отсутствии расширения — падение старта с понятным сообщением. Не полагаться только на «миграции накатились».

7. **Расположение e2e теста** — Принимаем структуру **`integration/e2e/`** (или `test/e2e/`): один или несколько `*_test.go`, которые поднимают БД (testcontainers), накатывают миграции, поднимают HTTP-сервер и проходят сценарий из README. Запуск: `go test -tags=integration ./integration/e2e/...` (или `./test/e2e/...`).

8. **Модульный путь (github.com/yourusername/project)** — В рамках 08 **оставить как есть**; замена на фактический путь репозитория — отдельная задача или при первом push. Если в проекте уже зафиксирован другой module path — следовать ему.

## Требования
- Единая инициализация:
  - подключение к Postgres
  - проверка, что расширение pgvector доступно (на старте или миграциями)
- Регистрация маршрутов строго по OpenAPI:
  - несоответствия должны ловиться contract tests
- Конфиг:
  - DSN Postgres
  - секрет для HMAC токенов (если выбран HMAC)
  - дефолтные значения (top_k, threshold, лимиты шаблонов)

## Минимальный e2e сценарий (обязателен)
E2E сценарий из `docs/agents/README.md` должен быть реализован как тест:
1) Создать app + app token (staff).
2) Создать staff token.
3) Создать кейс draft → pending_review → approved (embedding появляется).
4) Вызвать search app токеном, получить chunks, при низком confidence получить ticket.
5) Обработать тикет и convert-to-case, получить draft кейс.

## Критерии приёмки
- Сервис запускается локально “одной командой” (как минимум для интеграционных тестов).
- E2E тест стабильно проходит на чистой БД.

---

## Отчёт о реализации (handoff)

### Созданные и изменённые файлы

**Новые:**
- `internal/auth/sqlstore.go` — реализация TokenStore на pgxpool
- `internal/embedding/stub.go` — заглушка EmbeddingProvider (вектор размерности 1536)
- `internal/config/config.go` — конфиг (DSN, secret, TicketsBaseURL, TemplateMaxOutputLen)
- `internal/glue/case_creator.go` — адаптер tickets.CaseCreator через cases.Service.Create
- `internal/http/router.go` — роутер и цепочка Metrics → Auth → EnrichPrincipal → маршруты
- `cmd/api/main.go` — точка входа: Postgres, миграции, проверка pgvector, сборка зависимостей и запуск HTTP
- `integration/e2e/e2e_test.go` — минимальный e2e (app/staff токены, кейс draft→approved, search, тикет, convert-to-case)

**Изменённые:**
- `internal/auth/sqlstore_test.go` — переход на NewSQLTokenStore и pgxpool, миграции через runMigrationPool по connStr

### Запуск сервиса локально

- Postgres с расширением pgvector (например, образ `pgvector/pgvector:pg16`).
- Переменные окружения:
  - `DATABASE_URL` или `POSTGRES_DSN` — DSN (обязательно)
  - `AUTH_SECRET` или `HMAC_SECRET` — секрет для HMAC (по умолчанию `dev-secret`)
  - `LISTEN_ADDR` — адрес слушать (по умолчанию `:8080`)
- Запуск из корня репозитория:
  ```bash
  export DATABASE_URL="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
  go run ./cmd/api
  ```
  Либо после сборки: `./api` (если бинарник собран в текущую директорию).
- Миграция `db/migrations/0001_init.sql` накатывается при старте. После миграций выполняется проверка наличия расширения pgvector; при его отсутствии процесс падает с понятной ошибкой.

### Запуск тестов

- Обычные тесты (без интеграции): `go test ./...`
- С интеграцией и e2e (Docker обязателен): `go test -tags=integration ./...`
- Только e2e: `go test -tags=integration ./integration/e2e/...`

### Что сделано по пунктам

- **Postgres и миграции** — в main и e2e: пул, накат `0001_init.sql`, проверка pgvector после миграций.
- **Маршруты по OpenAPI** — POST `/api/v1/search` (RequireAppOrStaff), `/api/v1/cases`, `/api/v1/tickets`, `/api/v1/apps` (RequireStaff).
- **Цепочка middleware** — Metrics(writer) → Authenticate(store, secret) → EnrichPrincipal → маршруты с RequireStaff/RequireAppOrStaff.
- **Embedding и шаблоны** — stub в `internal/embedding`, рендерер из `internal/template` с лимитом из конфига.
- **E2E** — создание app + app/staff токенов в БД, кейс draft → pending_review → approved, search (при низком confidence — тикет из ответа, иначе создаётся вручную), convert-to-case, проверка появления draft-кейса в списке.
