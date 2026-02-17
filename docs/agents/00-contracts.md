# Модуль 00 — Contracts (OpenAPI + миграции БД + схемы/примеры)

## Промпт для изолированного агента (копипаст)

```text
Ты агент-разработчик. Твоя задача — реализовать Модуль 00 (Contracts): заморозить контракты интеграции (OpenAPI + миграции БД + примеры), чтобы остальные модули могли разрабатываться независимо.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/00-contracts.md.
- Стек: Go + PostgreSQL + pgvector + PostgreSQL FTS. Auth: Authorization: Bearer <token>, token_type: app/staff.
- Embeddings храним только для cases.status=approved.
- confidence = cosine similarity.

Ограничения по изменениям (важно):
- ТЕБЕ МОЖНО создавать/править только контракты и схему:
  - contracts/openapi.yaml
  - contracts/examples/*
  - contracts/jsonschema/* (если нужно)
  - db/migrations/*
- Не меняй бизнес-логику модулей (код), если она уже есть в репозитории. Если кода нет — не создавай его здесь, только контракты/миграции/примеры.

Что нужно сделать (порядок):
1) Сформируй OpenAPI (contracts/openapi.yaml) для всех эндпоинтов из docs/main_doc (search, cases, tickets, apps/settings).
2) Зафиксируй единый error shape и примеры 401/403/404/409/422/500.
3) Подготовь миграции Postgres:
   - cases (+ search_tsv + GIN индекс)
   - case_embeddings (pgvector, cosine ops, один embedding на кейс, только approved)
   - tickets, apps, auth_tokens, request_metrics
4) Добавь “golden examples” JSON для каждого эндпоинта: success + validation error (и auth error где применимо).
5) Проверь, что миграции применяются на чистую БД (Postgres + pgvector).
6) Итогом дай краткую инструкцию: как валидировать OpenAPI и как прогнать миграции.

Definition of Done:
- OpenAPI валиден, схемы/примеры соответствуют.
- Миграции накатываются на чистую БД.
- Примеры покрывают все эндпоинты и базовые ошибки.

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- перечисли команды/шаги проверки (валидация openapi + применение миграций).
```

## Цель
Создать и “заморозить” контракты, по которым будут интегрироваться остальные модули без синхронизаций:
- HTTP контракты (OpenAPI)
- схема данных и индексы (миграции)
- форматы JSON (настройки, импорт/экспорт) и примеры

## Выходные артефакты (обязательно)
- `contracts/openapi.yaml`
- `contracts/examples/*.json` (минимум по 1 примеру на эндпоинт + примеры ошибок)
- `db/migrations/0001_init.sql` и последующие миграции при необходимости
- (опционально) `contracts/jsonschema/app_settings.json`

## OpenAPI: что должно быть описано
### Эндпоинты (полный перечень)
- `POST /api/v1/search`
- Cases:
  - `GET /api/v1/cases`
  - `POST /api/v1/cases`
  - `GET /api/v1/cases/{id}`
  - `PUT /api/v1/cases/{id}`
  - `DELETE /api/v1/cases/{id}`
  - `POST /api/v1/cases/{id}/status`
  - `POST /api/v1/cases/import`
  - `GET /api/v1/cases/export`
- Tickets:
  - `GET /api/v1/tickets`
  - `POST /api/v1/tickets`
  - `GET /api/v1/tickets/{id}`
  - `PUT /api/v1/tickets/{id}`
  - `POST /api/v1/tickets/{id}/convert-to-case`
- Apps/Settings:
  - `GET /api/v1/apps`
  - `POST /api/v1/apps`
  - `GET /api/v1/apps/{id}`
  - `PUT /api/v1/apps/{id}`
  - `GET /api/v1/apps/{id}/settings`
  - `PUT /api/v1/apps/{id}/settings`
  - `GET /api/v1/apps/{id}/settings/export`
  - `POST /api/v1/apps/{id}/settings/import`

### Auth и права доступа
- `bearerAuth` (Bearer токен)
- Описание прав:
  - `search`: app + staff
  - остальное: staff
- Важно: фиксировать, какие операции требуют “создатель draft”.

### Схемы запросов/ответов (ключевое)
- `SearchRequest`: `query` (required), `category` (optional), `top_k` (optional), `user_context` (optional object).
- `SearchResponse`: `chunks[]` + optional `ticket`.
- `Chunk`: `case_id`, `title`, `text`, `confidence` (float).
- `TicketRef`: `id`, `url`.
- `Case`: `id`, `category`, `title`, `questions[]`, `keywords[]`, `response_template`, `status`, `created_by`, timestamps.
- `StatusChangeRequest`: `status`, optional `comment`.
- `Ticket`: `id`, `query`, `category`, `confidence`, `status`, `assigned_to`, timestamps, `resolution_notes`, `converted_to_case_id`.
- `App`: `id`, `name`, `settings`.

### Ошибки (единый формат)
В OpenAPI обязателен единый error schema, примеры 401/403/404/409/422/500.

## Миграции БД: что должно быть создано
Минимум таблиц:
- `cases` + `search_tsv`
- `case_embeddings` (pgvector)
- `tickets`
- `apps`
- `auth_tokens`
- `request_metrics`

Обязательные инварианты:
- В `case_embeddings` должна быть 1 запись на кейс, и записи существуют **только для `approved`**.
- Индекс под cosine similarity должен быть явно указан/закомментирован (через `vector_cosine_ops`).
- FTS: GIN индекс на `cases.search_tsv`.

## Примеры (обязательные)
Для каждого эндпоинта минимум:
- success response
- validation error
- unauthorized/forbidden (если применимо)

## Тесты и приёмка
- OpenAPI валидируется стандартным валидатором.
- Миграции применяются на чистую БД (Postgres с pgvector).
- Примеры соответствуют схемам OpenAPI (проверяется в contract tests).

