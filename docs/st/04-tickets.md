# Модуль 04 — Tickets (CRUD + workflow + convert-to-case)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 04 (Tickets): staff API для тикетов + workflow + convert-to-case.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/04-tickets.md.
- Контракты и миграции задаёт Модуль 00 (contracts/openapi.yaml и db/migrations/*). Считай их “замороженными”.
- Auth и principal в context даёт модуль 01.
- convert-to-case должен создать draft кейс и связать его (через интерфейс CaseCreator/CasesWriter, определяемый в 04; реализация — в Glue из 03).
- Тело convert-to-case: OpenAPI ConvertToCaseRequest (title, category, response_template опциональны); дефолты — см. секцию «Уточнения».
- POST /api/v1/tickets реализовать (ручное создание staff); фильтр category по полю из БД.

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
   - прочие поля (title/category/response_template) — из тела запроса или дефолты (см. Уточнения ниже)
2) записать `tickets.converted_to_case_id`
3) вернуть ссылку на кейс/его id (формат ответа — OpenAPI `ConvertToCaseResponse`: case_id, url)

Примечание: embedding для draft не создаётся (это правило Cases модуля).

## Уточнения (решения по открытым вопросам)

1. **Контракт convert-to-case и тело запроса**  
   В OpenAPI: `ConvertToCaseRequest` с **опциональными** полями `title`, `category`, `response_template`. Реализация: при вызове convert-to-case читать тело по схеме OpenAPI. **Дефолты при отсутствии полей:** `title` — из `ticket.query` (обрезать до разумной длины, например 200 символов) или строка `"From ticket"`; `category` — из `ticket.category` или `"general"`; `response_template` — минимальный шаблон, например `"{{.Query}}"` или константа-плейсхолдер. Все поля кейса должны быть валидны (NOT NULL и т.д. по миграциям).

2. **Интерфейс CasesWriter (CaseCreator)**  
   **Определяется в модуле 04** (например `internal/tickets` или `internal/tickets/convert.go`), чтобы 04 не импортировал 03. Сигнатура — один метод создания draft-кейса, например:
   - `CreateDraftFromTicket(ctx context.Context, req CreateDraftFromTicketRequest) (caseID string, err error)`
   - где `CreateDraftFromTicketRequest` содержит: `Query` (из тикета), `Title`, `Category`, `ResponseTemplate` (все из тела запроса или дефолты выше), `CreatedBy` (principal.TokenID). Реализацию интерфейса в Glue передаёт модуль 03 (или обёртка над cases.Service/CreateDraft). Код 03 не меняем; при сборке в 08 подставляется адаптер.

3. **Создание тикета из Search и ручное создание**  
   Тикеты создаются (a) потоком Search (02) при low confidence, (b) **ручным** `POST /api/v1/tickets` (staff). Оба сценария входят в зону 04: 04 реализует и чтение/обновление тикетов, и ручное создание по OpenAPI. Интеграционный тест может проверять: ручное создание тикета → convert-to-case → draft кейс; при наличии связи с 02 — сценарий «тикет из Search → обновление → convert-to-case».

4. **Principal в контексте**  
   Да: в хендлерах тикетов только **читать principal из контекста** (middleware модуля 01) и передавать его (например `principal.TokenID`) в `created_by` при создании кейса в convert-to-case. Своей логики авторизации в 04 нет; доступ к эндпоинтам тикетов — только staff (проверка в middleware до вызова 04).

5. **Фильтр category в GET /api/v1/tickets**  
   В миграции в таблице `tickets` есть колонка `category` (VARCHAR(100)) и индекс `idx_tickets_category`. Семантика и наличие полей заданы OpenAPI и миграциями. Реализовать фильтр по `category` в списке тикетов: при передаче query-параметра `category` — фильтровать `WHERE category = $1` (или совпадение по значению из OpenAPI).

6. **Опора на контракты**  
   При реализации опираться на `contracts/openapi.yaml` и `db/migrations/*` для полей и схем. Интерфейсы Cases/Search использовать только по контракту, код модулей 02/03 не менять.

## Внутренние интерфейсы (для изоляции)
- `TicketsRepository` (CRUD по БД)
- **CaseCreator** (или `CasesWriter`): интерфейс с методом `CreateDraftFromTicket(ctx, req) (caseID string, err error)` — **определяется в 04**, реализация инжектируется в Glue (из 03 или адаптер)

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

