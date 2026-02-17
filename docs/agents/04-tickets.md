# Модуль 04 — Tickets (CRUD + workflow + convert-to-case)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 04 (Tickets): staff API для тикетов + workflow + convert-to-case.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/04-tickets.md.
- Контракты и миграции задаёт Модуль 00 (contracts/openapi.yaml и db/migrations/*). Считай их “замороженными”.
- Auth и principal в context даёт модуль 01.
- convert-to-case должен создать draft кейс и связать его (через Cases интерфейс/репозиторий).

Ограничения по изменениям (важно):
- Не меняй contracts/openapi.yaml и db/migrations/* (это зона Модуля 00).
- Не меняй Search/Cases/Apps/Auth, только используй интерфейсы.

Куда класть код (если структура проекта ещё не задана):
- internal/tickets/* (service/use-cases)
- internal/tickets/http/* (handlers для /api/v1/tickets)
- internal/tickets/repository/* (SQL)
Если структура проекта уже есть — следуй ей.

Что нужно сделать (порядок):
1) Реализуй CRUD /api/v1/tickets по OpenAPI (list/filter, get, update).
2) Реализуй workflow статусов: open/in_progress/resolved/closed (валидация).
3) Реализуй convert-to-case:
   - создать draft кейс с questions=[ticket.query], created_by=principal
   - записать converted_to_case_id в ticket
4) Покрой unit/integration/contract тестами:
   - создание тикета из Search (можно через интеграционный сценарий)
   - конвертация в кейс и права доступа к созданному draft

Definition of Done:
- Полный цикл: ticket → update → convert-to-case работает и тестируется.
- Ошибки и ответы соответствуют OpenAPI и единому error shape.

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- как запустить тесты;
- какие решения приняты по полям, требуемым для convert-to-case (что заполняется дефолтом, что требовать во входе).
```

## Цель
Реализовать staff API для тикетов:
- list/filter
- получение тикета
- обновление статуса/назначения/заметок
- `convert-to-case`: создать `draft` кейс на основе тикета и связать его

## Границы модуля
Модуль не реализует поиск (02) и CRUD кейсов (03), но должен уметь:
- создавать кейс при конвертации через интерфейс “CasesWriter” (внутренний)

## Данные
Таблица `tickets`:
- `query` хранится здесь (это единственное место, где сохраняем текст запроса пользователя)
- `status`: `open|in_progress|resolved|closed`
- `assigned_to`, `resolution_notes`, `converted_to_case_id`

## Контракт (HTTP)
Полностью следует OpenAPI из модуля 00.

Эндпоинты:
- `GET /api/v1/tickets` (фильтры: status, category, даты)
- `POST /api/v1/tickets` (опционально, ручное создание staff)
- `GET /api/v1/tickets/{id}`
- `PUT /api/v1/tickets/{id}`
- `POST /api/v1/tickets/{id}/convert-to-case`

## convert-to-case (обязательная логика)
При `POST /api/v1/tickets/{id}/convert-to-case`:
1) создать кейс:
   - `status=draft`
   - `questions=[ticket.query]`
   - `created_by = principal`
   - прочие поля (title/category/response_template) — минимально валидные дефолты либо требовать input (фиксируется OpenAPI)
2) записать `tickets.converted_to_case_id`
3) вернуть ссылку на кейс/его id

Примечание: embedding для draft не создаётся (это правило Cases модуля).

## Внутренние интерфейсы (для изоляции)
- `TicketsRepository` (CRUD по БД)
- `CasesWriter` (создать draft кейс для конвертации)

## Тесты
### Unit
- валидация статусных полей
- корректное обновление `resolution_notes`, `assigned_to`

### Integration
- тикет создаётся из Search (02) → читается/обновляется здесь
- convert-to-case: появляется новый `draft` кейс и корректно проставляется `converted_to_case_id`

### Contract
- соответствие схемам OpenAPI

## Критерии приёмки
- Можно замкнуть цикл: low confidence → ticket → resolved → convert-to-case → draft кейс.

