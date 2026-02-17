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

---

## Минимальный e2e сценарий (должен пройти после интеграции)
1) Staff создаёт app и app token.
2) Staff создаёт staff token.
3) Staff создаёт кейс (`draft`) → отправляет на ревью (`pending_review`) → одобряет (`approved`) → embedding появляется в `case_embeddings`.
4) App вызывает `POST /api/v1/search`:
   - возвращаются `chunks` (учитывается `top_k`)
   - при `confidence < threshold` создаётся `ticket`, и `ticket{id,url}` возвращается
5) Staff обрабатывает тикет → `convert-to-case` → появляется новый `draft` кейс у создателя.

