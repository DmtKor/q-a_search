# Модуль 01 — Auth & Access Control (Bearer tokens)

## Промпт для изолированного агента

```text
Ты агент-разработчик. Твоя задача — реализовать Модуль 01 (Auth & Access Control): единый Bearer-token auth и базовые правила авторизации.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/01-auth.md.
- Контракты и миграции задаёт Модуль 00 (contracts/openapi.yaml и db/migrations/*). Считай их “замороженными”.

Ограничения по изменениям (важно):
- Не меняй contracts/openapi.yaml и db/migrations/* (это зона Модуля 00).
- Не меняй код других модулей, кроме Auth.

Куда класть код (если структура проекта ещё не задана):
- internal/auth/* (проверка токена, principal)
- internal/http/middleware/auth/* (HTTP middleware, если есть отдельный слой)
Если структура проекта уже есть — следуй ей и не создавай параллельные “вторые” каталоги.

Что нужно сделать (порядок):
1) Реализуй разбор заголовка Authorization: Bearer <token>.
2) Реализуй проверку токена по таблице auth_tokens (хранится только token_hash).
3) Реализуй principal в context: token_id, token_type (app/staff), role?, app_id?.
4) Реализуй декларативные проверки:
   - RequireAppOrStaff
   - RequireStaff
5) Реализуй базовую проверку “draft только создателю” как helper (используется в Cases).
6) Сделай unit + integration тесты (expires, disabled, 401/403).

Definition of Done:
- 401/403 различаются корректно, формат ошибок соответствует OpenAPI error schema.
- Тесты покрывают основные ветки.
- Компонент можно подключить к любому handler’у без копипасты проверок.

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- как запустить unit/integration тесты;
- какие решения приняты (HMAC vs SHA-256) и где задан секрет.
```

## Цель
Реализовать единый механизм аутентификации/авторизации:
- `Authorization: Bearer <token>`
- тип токена `app` или `staff`
- базовые правила доступа к эндпоинтам и ресурсам (в т.ч. “draft только создателю”)

## Границы модуля
В модуле **нет** бизнес-логики кейсов/поиска/тикетов. Только:
- проверка токена
- формирование principal в контексте запроса
- политики доступа (эндпоинт/ресурс)

## Данные
Таблица `auth_tokens`:
- `token_hash` (в БД только хеш)
- `token_type`: `app|staff`
- `role` (на будущее)
- `app_id` (если token_type=app)
- `is_active`, `expires_at`, `last_used_at`

## Контрактные решения (фиксируем)
### Хеширование токена
Предпочтительно:
- `token_hash = HMAC-SHA-256(server_secret, raw_token)`

Допустимо (если нет секрета):
- `token_hash = SHA-256(raw_token)` (хуже по безопасности).

### Ошибки
- Нет/невалидный токен: `401 unauthorized`
- Токен валиден, но нет доступа: `403 forbidden`

## API (внутренний интерфейс)
Модуль обязан предоставить удобные функции/мидлвари:
- `Authenticate(next)` → кладёт principal в context
- `RequireAppOrStaff(next)`
- `RequireStaff(next)`

И helper для проверок “создатель ресурса”:
- `IsOwner(principal, resource_created_by) bool`

## Политики доступа (MVP)
- `POST /api/v1/search`: app + staff
- `GET/POST/PUT/DELETE /api/v1/cases...`: staff
  - дополнительно: операции над `draft` разрешены только создателю (если статус `draft`)
- `tickets`, `apps`: staff

## Тесты
### Unit
- корректный разбор `Authorization` (включая пробелы/регистр)
- отсутствие токена → 401
- неверный токен → 401
- staff-only эндпоинт с app токеном → 403

### Integration
- токен в БД активен/неактивен
- `expires_at` истёк → 401
- `last_used_at` обновляется (если это требование включено)

## Критерии приёмки
- Любой хендлер может декларативно “навесить” требование `staff` или `app+staff`.
- Ошибки полностью соответствуют error schema из OpenAPI.

