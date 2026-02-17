# Модуль 03 — Cases (CRUD + status + embeddings lifecycle + import/export)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 03 (Cases): staff CRUD для кейсов, endpoint смены статуса, import/export, FTS tsv, и жизненный цикл embeddings (только approved).

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/03-cases.md.
- Контракты и миграции задаёт Модуль 00 (contracts/openapi.yaml и db/migrations/*). Считай их “замороженными”.
- Auth и principal в context даёт модуль 01 (нужен для “draft только создателю”).
- Embedding provider нужен только при переходе pending_review → approved.

Ограничения по изменениям (важно):
- Не меняй contracts/openapi.yaml и db/migrations/* (это зона Модуля 00).
- Не меняй Search/Tickets/Apps/Auth, только используй интерфейсы.

Куда класть код (если структура проекта ещё не задана):
- internal/cases/* (use-cases/service)
- internal/cases/http/* (handlers для /api/v1/cases)
- internal/cases/repository/* (SQL)
- internal/cases/fts/* (построение search_tsv)
Если структура проекта уже есть — следуй ей.

Что нужно сделать (порядок):
1) Реализуй CRUD /api/v1/cases по OpenAPI (POST создаёт draft, PUT не меняет status).
2) Реализуй endpoint POST /api/v1/cases/{id}/status с переходами из требований.
3) Реализуй правила доступа:
   - draft: GET/PUT/DELETE только создатель (created_by == principal)
4) Реализуй embedding lifecycle:
   - при pending_review → approved: вычислить embedding по keywords+questions и upsert в case_embeddings
   - при уходе из approved: удалить embedding
   - для draft/pending embeddings в БД не хранить
5) Реализуй заполнение cases.search_tsv при POST/PUT.
6) Реализуй import/export, включая replace (удалить и заменить только cases+embeddings).
7) Покрой unit/integration/contract тестами, включая pgvector/FTS.

Definition of Done:
- Все правила статусов, доступа и embeddings соблюдены.
- replace работает как “откат”: заменяет cases и чистит embeddings, не трогая tickets/apps/tokens/metrics.
- Интеграционные тесты проходят на чистой БД.

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- как запустить тесты;
- какие edge-cases учтены (например повторное approve, откат, конфликт обновлений).
```

## Цель
Реализовать staff API для кейсов:
- CRUD `/api/v1/cases`
- смена статуса `POST /api/v1/cases/{id}/status`
- import/export
- соблюдение правил доступа (draft только создателю)
- жизненный цикл embedding: **в БД хранится только для `approved`**
- поддержка FTS: заполнение `cases.search_tsv` на записи/обновлении

## Границы модуля
Модуль не реализует:
- `POST /api/v1/search` (модуль 02)
- тикеты (модуль 04)
- auth (модуль 01)
- настройки (модуль 05)

## Данные
Таблицы:
- `cases`
- `case_embeddings`

Инварианты:
- `case_embeddings` содержит запись только если `cases.status='approved'`.
- Один embedding на один кейс.

## Правила доступа (MVP)
Для staff principal:
- Если `case.status='draft'`:
  - `GET /cases/{id}` и `PUT /cases/{id}` разрешены только если `created_by == principal`
  - `DELETE /cases/{id}`: решение MVP — только создатель
- Если статус не draft: доступ staff (без детализации ролей в MVP)

## CRUD поведение
- `POST /api/v1/cases`:
  - создаёт `draft`
  - выставляет `created_by`
  - заполняет `search_tsv`
- `PUT /api/v1/cases/{id}`:
  - запрещено менять `status` напрямую (только через status endpoint)
  - пересчитывает `search_tsv`
  - если кейс в `approved` и изменились `keywords/questions/title/response_template`:
    - embedding lifecycle управляется по правилам ниже (см. статус)

## Смена статуса (MVP)
Эндпоинт: `POST /api/v1/cases/{id}/status`

Разрешённые переходы:
- создатель: `draft → pending_review`
- staff: `pending_review → approved`
- staff: `pending_review → draft` (с комментарием)
- staff: `approved → archived`

### Embedding lifecycle при смене статуса
- `pending_review → approved`:
  - сервер вычисляет embedding по **`keywords + questions`** (один вектор на кейс)
  - upsert в `case_embeddings`
- любой переход **из** `approved` (например `approved → archived`):
  - удалить `case_embeddings` по `case_id`

Важно:
- Для `draft/pending_review` embedding в БД **не хранить** (для тестов рассчитывается на стороне клиента).

## Import/Export
### Export
- Возвращает JSON массив кейсов по фильтрам (category/status), формат фиксируется в OpenAPI.

### Import
`POST /api/v1/cases/import?mode=merge|draft|replace`
- `merge`: обновить существующие по `id`, остальные создать
- `draft`: все импортированные приводятся к `draft`
- `replace`: **удалить все кейсы** и заменить импортом

Уточнение по `replace` (фиксируем):
- `replace` затрагивает только `cases` и `case_embeddings`.
- `tickets`, `apps`, `auth_tokens`, `request_metrics` не трогаются.

## FTS: заполнение `search_tsv`
Модуль обязан поддерживать формирование `search_tsv` на `POST/PUT`.
Минимум: включать `title`, `keywords`, `questions` (как текст).
Язык/словарь можно оставить дефолтным (уточнить позже).

## Тесты
### Unit
- доступ к draft только создателю
- запрет смены статуса через `PUT` (только status endpoint)
- проверка разрешённых переходов статуса

### Integration (Postgres + pgvector + FTS)
- `approved` → запись в `case_embeddings` появляется
- `approved → archived` → запись исчезает
- `draft/pending_review` → embedding не создаётся
- `search_tsv` заполнен и обновляется
- `replace` реально удаляет/заменяет `cases` и чистит `case_embeddings`

### Contract
- ответы/ошибки строго по OpenAPI

## Критерии приёмки
- Все инварианты статусов и embeddings соблюдаются.
- Импорт/экспорт совместимы между версиями модулей (контракт “заморожен” в 00).

