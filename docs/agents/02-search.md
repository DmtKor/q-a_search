# Модуль 02 — Search (Hybrid: pgvector + FTS, cosine confidence, tickets)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 02 (Search): POST /api/v1/search (hybrid pgvector+FTS) с confidence=cosine similarity и созданием ticket при низком confidence.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/02-search.md.
- Контракты и миграции задаёт Модуль 00 (contracts/openapi.yaml и db/migrations/*). Считай их “замороженными”.
- Auth уже решён модулем 01 (principal в context, token_type app/staff).
- Настройки threshold/default_top_k отдаёт модуль 05 (Apps settings).
- Рендер шаблонов отдаёт модуль 06 (Template).

Ограничения по изменениям (важно):
- Не меняй contracts/openapi.yaml и db/migrations/* (это зона Модуля 00).
- Не меняй реализацию Cases/Tickets/Apps/Auth, только используй их через интерфейсы.

Куда класть код (если структура проекта ещё не задана):
- internal/search/* (core логика поиска)
- internal/search/http/* (handler для POST /api/v1/search, если HTTP слой живёт внутри модуля)
- internal/search/repository/* (SQL для retrieval)
Если структура проекта уже есть — следуй ей.

Что нужно сделать (порядок):
1) Реализуй handler POST /api/v1/search по OpenAPI.
2) Реализуй гибридный retrieval:
   - pgvector (cosine) по case_embeddings (только approved)
   - FTS по cases.search_tsv (только approved)
   - объединение/реранк (формула не фиксируется контрактом)
3) Зафиксируй, что chunks[].confidence = cosine similarity (векторный скор), даже если ранжирование гибридное.
4) Реализуй top_k и дефолты (через модуль 05) + валидируй диапазон.
5) При confidence(top1) < threshold: создай ticket (через модуль 04 или репозиторий тикетов) и верни ticket{id,url}.
6) Для каждого chunk отрендери response_template (через модуль 06).
7) Покрой unit/integration/contract тестами:
   - не возвращаем draft/pending/archived
   - category filter
   - ticket создаётся только на low confidence
   - confidence именно cosine

Definition of Done:
- JSON ответы соответствуют OpenAPI.
- Интеграционные тесты с реальным Postgres+pgvector+FTS проходят.
- Нет “рассинхрона” с ticket (если вернули ticket — он реально создан).

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- как запустить тесты;
- какие параметры (top_k min/max, default threshold) использованы по умолчанию.
```

## Цель
Реализовать `POST /api/v1/search`:
- Гибридный retrieval: **pgvector + PostgreSQL FTS**
- Фильтрация: искать **только по `cases.status=approved`**
- `confidence` в `chunks[]`: **cosine similarity**
- Возвращать несколько `chunks` (управляется `top_k`)
- При `confidence < threshold` создавать `ticket` и возвращать `ticket{id,url}`

## Границы модуля
Модуль не отвечает за:
- CRUD кейсов/тикетов (это модули 03/04)
- проверку токенов (это модуль 01)
- хранение/редактирование настроек приложения (это модуль 05)

Но модуль должен уметь:
- прочитать применимые настройки (threshold/default_top_k) через интерфейс
- создать тикет через интерфейс

## Контракт (HTTP)
Полностью следует OpenAPI из модуля 00.

Ключевые инварианты:
- если `category` отсутствует → поиск по всем категориям
- если `category` задана → фильтр по категории
- `chunks[].confidence` — именно cosine similarity (даже если ранжирование гибридное)

## Данные и запросы
Используются таблицы:
- `cases` (approved-only, + `search_tsv`)
- `case_embeddings` (только для approved)
- `tickets` (создание при low confidence)

## Внутренние интерфейсы (для изоляции и тестируемости)
Модуль должен быть написан через интерфейсы:

- `EmbeddingProvider`
  - `EmbedQuery(ctx, query string) ([]float32, error)`

- `SearchRepository`
  - `SearchApproved(ctx, params) ([]Candidate, error)`
  - Candidate содержит `case_id`, `title`, `response_template`, `category`, `cosine_similarity`, FTS rank (если есть)

- `TemplateRenderer`
  - `Render(ctx, template string, userContext map[string]any) (string, error)`

- `TicketsWriter`
  - `CreateLowConfidenceTicket(ctx, data) (ticketID string, ticketURL string, error)`

- `AppSettingsReader`
  - `GetEffectiveSearchSettings(ctx, principal) (threshold float64, defaultTopK int, err error)`

## Уточнения (решения по открытым вопросам)
Закреплено для согласованной реализации и интеграции с 04/05/08.

1. **AppSettingsReader при staff без app_id**  
   Интерфейс только `GetEffectiveSearchSettings(ctx, principal)`. Реализация (модуль 05) при отсутствии app (например staff-токен без app_id) возвращает дефолты (например threshold=0.7, defaultTopK=10). Модуль 02 только вызывает интерфейс; в тестах мокаем любые значения.

2. **Формула объединения vector + FTS**  
   Берём по `2*top_k` кандидатов из vector и из FTS, объединяем по `case_id` (при дубликате оставляем запись с лучшим cosine), реранжируем по формуле (например RRF или `0.5*норм_cosine + 0.5*норм_fts_rank`), затем берём `top_k`. В ответе у каждого chunk по-прежнему **confidence = cosine similarity** (векторный скор).

3. **TicketRef.url**  
   Формат URL не задаётся в 02. Формирует тот, кто реализует TicketsWriter (модуль 04/Glue); модуль 02 только подставляет в ответ возвращённый `ticketURL`. В тестах мок может возвращать, например, `"/api/v1/tickets/" + id`.

4. **Данные для CreateLowConfidenceTicket**  
   В интерфейсе использовать структуру с полями: **Query** (string), **Category** (optional string), **Confidence** (float) — по полям таблицы `tickets`.

5. **search_tsv в тестах**  
   В БД `search_tsv` заполняется приложением при INSERT/UPDATE кейса (модуль 03). Модуль 02 только читает. В интеграционных тестах 02 в фикстурах явно проставлять `search_tsv` (например через `to_tsvector(...)` в SQL), чтобы FTS-тесты были предсказуемыми.

6. **Валидация top_k**  
   OpenAPI: minimum 1, maximum 50. Если в запросе не передан `top_k` — брать `defaultTopK` из настроек (AppSettingsReader). Итоговый `top_k_effective` ограничивать диапазоном [1, 50]; при значении вне диапазона возвращать **422** с `validation_error`.

## Алгоритм (обязательная логика)
1) Взять `query`, `category?`, `top_k?`.
2) Получить настройки: `threshold`, `defaultTopK`.
3) Определить `top_k_effective` (валидировать min/max).
4) Получить embedding запроса через `EmbeddingProvider`.
5) Выполнить retrieval:
   - vector candidates: cosine similarity (pgvector)
   - FTS candidates: ts_rank (FTS)
   - объединить/реранкнуть (формула не фиксируется контрактом)
6) Выбрать top_k.
7) `confidence` определить как cosine similarity кандидата (top1, и для каждого chunk — свой).
8) Если `confidence(top1) < threshold` → создать тикет, включить в ответ.
9) Для каждого chunk: отрендерить `response_template` с `user_context`.

## Тесты
### Unit
- без category → поиск по всем
- top_k применяется и валидируется (например min=1, max=50)
- ticket создаётся только при `confidence < threshold`
- `confidence` в ответе соответствует cosine similarity, а не гибридному ранку
- шаблон рендерится даже при отсутствующих полях `user_context` (через TemplateRenderer контракт)

### Integration (Postgres + pgvector + FTS)
- фикстуры: кейсы в разных статусах, embeddings только у approved
- проверка: draft/pending/archived не попадают в результаты
- проверка: фильтр по category работает
- проверка: FTS работает (по `search_tsv`)

### Contract
- ответы соответствуют OpenAPI схемам
- ошибки соответствуют error schema

## Критерии приёмки
- endpoint стабильно возвращает JSON по OpenAPI
- при создании тикета нет “рассинхрона” (если `ticket` заявлен в ответе — он реально создан)

