# Модуль 07 — Metrics (`request_metrics`, без хранения query)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 07 (Metrics): сбор request_metrics для всех HTTP запросов без хранения текста query.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/07-metrics.md.
- Контракты и миграции задаёт Модуль 00. Считай их “замороженными”.
- Auth (01) даёт principal/token_id/app_id (если доступно) — нужно прокинуть в метрики.
- Решения: endpoint = сырой путь (r.URL.Path); порядок Metrics → Auth → Handlers; обёртка ResponseWriter для status_code; интерфейс Writer, реализация в Glue; token_id/app_id — строка UUID или NULL. См. секцию «Уточнения».

Ограничения по изменениям (важно):
- Не меняй contracts/openapi.yaml и db/migrations/*.
- Не меняй бизнес-логику модулей — только middleware/писатель метрик.

Куда класть код (если структура проекта ещё не задана):
- internal/metrics/* (writer)
- internal/http/middleware/metrics/* (middleware)
Если структура проекта уже есть — следуй ей.

Что нужно сделать (порядок):
1) Реализуй middleware, который замеряет response_time_ms, status_code и endpoint.
2) Записывай request_metrics: endpoint, status_code, response_time_ms, token_id, app_id, created_at.
3) Обеспечь, что ошибки записи метрик не ломают основной ответ (best-effort).
4) Покрой unit/integration тестами (integration с реальной БД).

Definition of Done:
- Метрики пишутся для всех эндпоинтов, без query.
- Ошибки/лаг метрик не ломают API.

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- как запустить тесты;
- как формируется “endpoint” (сырой путь vs шаблон роута).
```

## Цель
Реализовать единый сбор метрик по HTTP запросам без хранения текста пользовательского `query`:
- endpoint
- status_code
- response_time_ms
- token_id
- app_id
- created_at

## Границы модуля
Модуль не отвечает за бизнес-логику и не должен влиять на ответы API, кроме как:
- оборачивать хендлеры middleware’ом
- безопасно писать метрики

## Требования
- Нельзя хранить текст `query` в метриках (он хранится только в `tickets` при low confidence).
- Запись метрик не должна существенно замедлять запрос:
  - MVP: синхронная запись допустима
  - предпочтительно: буфер/async writer (опционально)

## Уточнения (решения по открытым вопросам)

1. **Формат endpoint** — В MVP **сырой путь** `r.URL.Path`. В тестах и handoff зафиксировать. Позже при роутере с шаблонами — опция.

2. **Порядок middleware** — Метрики для всех запросов (включая 401/403). Цепочка: **Metrics → Auth → Handlers**. Metrics первым; при 401/403 principal nil → token_id/app_id NULL. В 08 регистрировать в этом порядке.

3. **status_code** — Оборачивать ResponseWriter, перехватывать WriteHeader(statusCode). Response wrapper.

4. **Writer** — Интерфейс `Write(ctx, record)` в internal/metrics; реализация с pgxpool; Glue создаёт и передаёт. 07 не зависит от cmd/.

5. **token_id, app_id** — В БД UUID; передавать строкой (UUID) или NULL при nil.

6. **Интеграционный тест** — testcontainers + db/migrations, вызов через middleware, проверка записи. Отдельной тест-миграции не нужно.

## Интерфейсы
- Middleware: `Metrics(writer, ...)(next)` — замер времени, обёртка ResponseWriter для status_code, в конце writer.Write(ctx, record).
- Writer: `Write(ctx, record)`; реализация с pgxpool в Glue.

## Тесты
### Unit
- endpoint = сырой путь (r.URL.Path); перехват status_code; в тестах фиксировать сырой путь.
- измерение response_time_ms

### Integration
- Postgres + миграции, вызов через middleware → запись в request_metrics; при 401 — запись есть, token_id/app_id NULL.

## Критерии приёмки
- Метрики пишутся для всех запросов (включая 401/403), без query.
- Ошибки записи метрик не ломают основной ответ.

