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

Куда класть код: только пакеты internal/auth и при необходимости internal/http/middleware/auth. Точку входа (cmd/main) не создавать — это зона модуля 08-glue.

Ключевые решения (подробно в теле документа):
- Секрет: переменная окружения AUTH_TOKEN_SECRET; хеш HMAC-SHA-256(secret, raw_token).
- 01 только проверка токена; создание токенов — другой модуль; для тестов — HashToken helper или вставка готового token_hash.
- Principal содержит token_id (UUID); cases.created_by = тот же token_id; IsOwner(principal, resource_created_by) — сравнение по token_id.
- last_used_at обновлять при каждой успешной аутентификации.
- net/http: middleware func(next http.Handler) http.Handler; principal в context.
- Доступ к БД только через интерфейс TokenStore (GetByTokenHash, при необходимости UpdateLastUsedAt); реализацию инжектирует Glue.

Что нужно сделать (порядок):
1) Определи интерфейс TokenStore и тип Principal (TokenID, TokenType, Role?, AppID?).
2) Реализуй разбор Authorization: Bearer <token>, вычисление hash, поиск через TokenStore.
3) Реализуй middleware Authenticate: principal в context, при успехе обновление last_used_at.
4) Реализуй RequireAppOrStaff, RequireStaff и helper IsOwner(principal, resource_created_by).
5) Сделай unit + integration тесты (expires, disabled, 401/403, last_used_at).

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
- Использовать **HMAC-SHA-256(server_secret, raw_token)**.
- **Секрет:** задаётся **переменной окружения** `AUTH_TOKEN_SECRET`. При старте приложения (в Glue/конфиге) читать её и передавать в компонент Auth. Конфиг-файл и флаги не используем для секрета.

### Создание токенов (вне зоны 01)
- **Модуль 01 реализует только проверку токена:** чтение `Authorization`, вычисление hash, поиск в `auth_tokens`, формирование principal.
- **Создание токенов** (генерация raw token, запись `token_hash` в БД) — зона другого модуля (Apps, Glue или отдельный admin). В 01 этого кода нет.
- **Для тестов и сидов:** либо вставлять в БД готовый `token_hash` (вычисленный заранее с тем же секретом), либо модуль 01 экспортирует **вспомогательную функцию** `HashToken(secret []byte, rawToken string) string` (или аналог) только для использования в сидах/тестах; создание записей в `auth_tokens` делают тесты/сиды через общий репозиторий.

### Ошибки
- Нет/невалидный токен: `401 unauthorized`
- Токен валиден, но нет доступа: `403 forbidden`

## Principal и «draft только создателю»
- В **principal** храним **идентификатор записи токена:** `token_id` (UUID из `auth_tokens.id`).
- В **`cases.created_by`** в MVP храним **тот же идентификатор** — строковое представление `auth_tokens.id` (т.е. «владелец» кейса = токен, от которого создан кейс).
- **IsOwner(principal, resource_created_by)** реализуется как сравнение: `principal.TokenID == resource_created_by` (оба строки, нормализовать формат при сравнении). Отдельного `user_id`/label в MVP не вводим; при появлении пользователей позже можно добавить в principal поле и маппинг token → user.

## last_used_at
- **В рамках 01-auth требование включено:** при каждой успешной аутентификации обновлять `last_used_at` у соответствующей записи в `auth_tokens`. Реализовать в 01 (в слое, который вызывает TokenStore после успешной проверки). Опционально позже можно сделать отключаемым (конфиг/флаг), если понадобится снизить нагрузку на запись.

## Структура проекта и граница модуля
- Модуль 01 **только поставляет пакеты:** `internal/auth`, при необходимости `internal/http/middleware/auth`. **Точки входа (cmd/server, main) и сборка приложения не входят в 01** — их делает модуль 08-glue. Тесты модуля 01 запускаются как обычные unit/integration тесты пакетов (без поднятия всего сервера); при необходимости интеграционные тесты поднимают только БД (например через testcontainers).

## HTTP-фреймворк и контекст
- Реализовать под **стандартный `net/http`:** middleware имеет вид `func(next http.Handler) http.Handler`, внутри вызывается `next.ServeHTTP` с контекстом, в который добавлен principal.
- Principal кладётся в `context.Context` через ключ типа (private type key), модуль экспортирует функции `PrincipalFromContext(ctx)` и, при необходимости, тип `Principal`. Для chi/gin/echo при сборке в Glue пишутся тонкие адаптеры (обёртки), которые извлекают контекст роутера и вызывают стандартный middleware; в 01 зависимостей от chi/gin/echo нет.

## Доступ к БД
- Модуль Auth **не принимает `*sql.DB` напрямую.** Он зависит от **интерфейса** (например `TokenStore`), с методом вроде `GetByTokenHash(ctx, hash string) (*TokenRow, error)` (и при необходимости `UpdateLastUsedAt(ctx, tokenID string) error`). Реализацию интерфейса (работа с `auth_tokens` через `*sql.DB`) делает Glue или общий слой репозиториев; в 01 только определение интерфейса и его использование. Явная рекомендация: **использовать только интерфейс** для тестируемости; в тестах подставляется мок.

## API (внутренний интерфейс)
Модуль обязан предоставить:
- **Middleware (net/http):** `Authenticate(TokenStore, secret []byte)(next http.Handler) http.Handler` — разбор Bearer, проверка через TokenStore, запись principal в context, при успехе всегда вызов UpdateLastUsedAt (метод в интерфейсе TokenStore может быть отдельным).
- `RequireAppOrStaff(next)`, `RequireStaff(next)` — обёртки, проверяющие principal из context и возвращающие 403 при несоответствии.
- **Helper:** `IsOwner(principal, resource_created_by string) bool` — сравнение по `principal.TokenID` и `resource_created_by`.
- **Тип Principal:** минимум поля `TokenID`, `TokenType` (app/staff), `Role`, `AppID` (опционально).

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
- при успешной аутентификации `last_used_at` обновляется

## Критерии приёмки
- Любой хендлер может декларативно “навесить” требование `staff` или `app+staff`.
- Ошибки полностью соответствуют error schema из OpenAPI.

