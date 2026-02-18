# Модульная разработка для изолированных агентов (индекс)

Цель: раздать разные части системы независимым агентам так, чтобы они могли **разработать и качественно протестировать** свои модули **без постоянной коммуникации**, а интеграция прошла по заранее “замороженным” контрактам.

Основание: `docs/main_doc` (Go + PostgreSQL + pgvector + PostgreSQL FTS, auth `Authorization: Bearer <token>`, токены `app/staff`, статусы `draft/pending_review/approved/archived`, embedding хранится **только для `approved`**, `confidence` = **cosine similarity**).

---

## Общие правила для всех модулей (обязательно)

### 1) Источники истины (контракты)
Интеграция строится вокруг артефактов, которые модуль 00 обязан создать и “заморозить”:

- **OpenAPI**: `contracts/openapi.yaml` (эндпоинты, схемы, примеры, коды ошибок, auth).
- **Миграции БД**: `db/migrations/*.sql` (PostgreSQL + pgvector + FTS).
- **JSON-настройки**: (опционально) JSON Schema для `apps.settings` и примеры.

Любой модуль считается готовым только если:
- проходит собственные unit/integration тесты;
- проходит **contract tests** против OpenAPI (валидность JSON, required поля, error shape).

### 2) Единый формат ошибок HTTP
Все ошибки возвращаются строго в формате:

```json
{
  "error": {
    "code": "string_machine_code",
    "message": "human-readable",
    "details": { "any": "json" }
  }
}
```

Минимальные `code`:
- `unauthorized`, `forbidden`
- `validation_error`
- `not_found`
- `conflict`
- `internal_error`

### 3) Security-инварианты
- **Bearer token**: `Authorization: Bearer <token>`
- Тип токена: **`app`** или **`staff`**
- Доступ:
  - `POST /api/v1/search`: app + staff
  - `/api/v1/cases`, `/api/v1/tickets`, `/api/v1/apps`: только staff
- **Draft** кейсы: по умолчанию видит/редактирует только **создатель**.

### 4) Тестовая инфраструктура
Каждый модуль поставляет:
- **Unit tests**: без БД (моки интерфейсов).
- **Integration tests**: с реальным PostgreSQL + pgvector + FTS (docker-compose или testcontainers).
- **Contract tests**: соответствие OpenAPI.

Рекомендация: `testcontainers-go` для поднятия Postgres с установленным `pgvector`.

---

## Модули (файлы требований)
- `docs/agents/00-contracts.md` — Contracts (OpenAPI + миграции + схемы/примеры)
- `docs/agents/01-auth.md` — Auth & Access Control
- `docs/agents/02-search.md` — Search (hybrid retrieval + cosine confidence + tickets)
- `docs/agents/03-cases.md` — Cases (CRUD + status + embeddings lifecycle + import/export)
- `docs/agents/04-tickets.md` — Tickets (CRUD + workflow + convert-to-case)
- `docs/agents/05-apps-settings.md` — Apps & Settings (JSON + import/export)
- `docs/agents/06-template-rendering.md` — Template Rendering (safe `text/template`)
- `docs/agents/07-metrics.md` — Metrics (`request_metrics`, без хранения query)
- `docs/agents/08-glue.md` — Glue (сборка сервера + минимальный e2e)
- `docs/agents/09-demo-ui.md` — Demo UI (тест/демо всех возможностей; без изменений в 00–08)

---

## Минимальный e2e сценарий (должен пройти после интеграции)
1) Staff создаёт app и app token.
2) Staff создаёт staff token.
3) Staff создаёт кейс (`draft`) → отправляет на ревью (`pending_review`) → одобряет (`approved`) → embedding появляется в `case_embeddings`.
4) App вызывает `POST /api/v1/search`:
   - возвращаются `chunks` (учитывается `top_k`)
   - при `confidence < threshold` создаётся `ticket`, и `ticket{id,url}` возвращается
5) Staff обрабатывает тикет → `convert-to-case` → появляется новый `draft` кейс у создателя.

---

## Как всё запустить и протестировать

Полное руководство для локального запуска API, UI и тестов.

### Требования

- **Go** (версия из `go.mod`)
- **Node.js** и **npm** (для Demo UI в `web/`)
- **Docker** (для Postgres с pgvector и для интеграционных/e2e тестов)

### 1. База данных

Postgres с pgvector — как в `docs/agents/00-contracts.md` (закреплённые команды):

```bash
docker run -d --name pgvector-test \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=kb \
  -p 5432:5432 \
  pgvector/pgvector:pg16
```

После запуска контейнера накатить миграции можно при старте API (они накатываются автоматически). Подключение: БД `kb`, пользователь `postgres`, пароль `postgres`, порт `5432`. DSN: `postgres://postgres:postgres@localhost:5432/kb?sslmode=disable`

### 2. API (бэкенд)

Из корня репозитория:

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/kb?sslmode=disable"
export AUTH_SECRET="dev-secret-change-in-production"   # опционально; это значение по умолчанию в конфиге
export LISTEN_ADDR=":8080"        # опционально
export REQUEST_LOG_LEVEL="minimal" # опционально: none | minimal | detailed (логи запросов/ответов)

go run ./cmd/api
```

- Миграции `db/migrations/0001_init.sql` накатятся при старте.
- После миграций выполняется проверка pgvector; при отсутствии расширения процесс завершится с ошибкой.
- API будет доступен по адресу `http://localhost:8080` (или выбранному `LISTEN_ADDR`).

### 3. Токены для ручного теста (UI и запросы к API)

В API нет эндпоинтов создания токенов. Токены нужно заранее положить в БД (таблица `auth_tokens`).

- **Секрет по умолчанию** (для проверки токенов): `dev-secret-change-in-production`. Задаётся переменной `AUTH_SECRET` или `HMAC_SECRET`; если не задана, API использует это значение (см. `internal/config/config.go`). Готового Bearer-токена по умолчанию нет — хотя бы один токен нужно создать в БД (см. ниже).
- **token_hash** в БД = hex(HMAC-SHA256(секрет, сырой_токен)). Секрет = переменная `AUTH_SECRET` (или `HMAC_SECRET`).
- В коде для сидов/тестов используется `auth.HashToken(secret, rawToken)` (пакет `internal/auth`).

**Вариант А — те же токены, что в e2e (удобно для воспроизведения):**

1. Задайте тот же секрет, что в e2e: `export AUTH_SECRET=e2e-secret`
2. Вставьте в БД приложение и токены с **сырыми** значениями `e2e-staff-token` (staff) и `e2e-app-token` (app). Хеш для каждой записи вычислите один раз в Go (например в тесте или маленькой утилите), вызвав `auth.HashToken([]byte("e2e-secret"), "e2e-staff-token")` и `auth.HashToken([]byte("e2e-secret"), "e2e-app-token")`, и подставьте полученный hex в `INSERT` в `auth_tokens`. Для app-токена нужна запись в `apps` (создайте через API под staff-токеном или вставьте вручную) и её `id` в `auth_tokens.app_id`.

**Вариант Б — свой токен:**

1. Придумайте сырой токен (например `my-staff-token`).
2. Вычислите хеш: в тесте или скрипте вызовите `auth.HashToken([]byte(os.Getenv("AUTH_SECRET")), "my-staff-token")` (при пустом env API использует `dev-secret-change-in-production` — тот же секрет подставьте при вычислении хеша).
3. Вставьте в `auth_tokens`: `token_hash`, `token_type` = `staff`, `is_active` = true. Для app-токена дополнительно нужна запись в `apps` и `app_id`.

**Вариант В — утилита seed-token (проще всего для локального теста):**

Из корня репозитория (при запущенном API и накатанных миграциях) выполните:
```bash
go run ./cmd/seed-token
```
Скрипт создаёт один staff-токен с сырым значением `local-staff-token` и выводит его в консоль. В Настройках UI введите этот токен. Используются те же `DATABASE_URL` и `AUTH_SECRET`, что и у API.

После этого в UI в Настройках укажите сырой токен и при необходимости базовый URL API (см. ниже).

### 4. Demo UI (веб-интерфейс)

```bash
cd web
npm install
npm run dev
```

- Обычно dev-сервер поднимается на `http://localhost:5173` (Vite).
- В UI откройте **Настройки**: введите **Bearer token** (сырой токен, например `e2e-staff-token` или `my-staff-token`) и при необходимости **Базовый URL API**.
- Если UI и API на одном хосте: оставьте URL пустым — запросы пойдут на `window.location.origin + '/api/v1'`. Если API на другом порту (например API на :8080, UI на :5173), укажите базовый URL явно, например `http://localhost:8080`.
- Нажмите «Сохранить в браузере». Тип доступа (App/Staff) определится автоматически при старте по запросу к API.

Дальше можно тестировать: Поиск, База знаний (Кейсы), Тикеты, Приложения — по спецификации `docs/agents/09-demo-ui.md`.

**Сборка UI для продакшена:** `npm run build` → артефакт в `web/dist/`. Раздавать можно отдельным веб-сервером или (опционально) маршрутом в API (модуль 08).

### 5. Тесты

- **Только unit/обычные тесты (без БД и Docker):**
  ```bash
  go test ./...
  ```

- **С интеграцией и e2e (нужен Docker):**
  ```bash
  go test -tags=integration ./...
  ```

- **Только e2e:**
  ```bash
  go test -tags=integration ./integration/e2e/...
  ```

E2e поднимает Postgres в testcontainers, накатывает миграции, создаёт app/staff токены, поднимает HTTP-сервер и прогоняет сценарий: кейс draft → approved, поиск, тикет, convert-to-case.

### 6. Краткий чеклист «всё с нуля»

1. Запустить Postgres с pgvector (команда из п. 1).
2. `export DATABASE_URL="postgres://postgres:postgres@localhost:5432/kb?sslmode=disable"; go run ./cmd/api` — дождаться старта API.
3. В **другом терминале** выполнить `go run ./cmd/seed-token` — создать staff-токен и скопировать из вывода значение (по умолчанию `local-staff-token`).
4. `cd web && npm install && npm run dev` — открыть UI в браузере (обычно http://localhost:5173).
5. В Настройках UI ввести Bearer token (из п. 3) и базовый URL API: `http://localhost:8080` (UI на 5173, API на 8080 — разные порты). Сохранить в браузере.
6. Проверить Поиск, База знаний (Кейсы), Тикеты, Приложения через UI.
7. При необходимости прогнать тесты: `go test ./...` и `go test -tags=integration ./...`.

