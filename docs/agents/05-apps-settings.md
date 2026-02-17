# Модуль 05 — Apps & Settings (JSON settings + import/export)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 05 (Apps & Settings): CRUD приложений и управление settings (JSON) с импортом/экспортом.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/05-apps-settings.md.
- Контракты и миграции задаёт Модуль 00 (contracts/openapi.yaml и db/migrations/*). Считай их “замороженными”.
- Search (02) будет читать threshold/default_top_k отсюда.
- Auth (01) даёт principal и для app токенов — app_id, по которому нужно найти settings.

Ограничения по изменениям (важно):
- Не меняй contracts/openapi.yaml и db/migrations/* (это зона Модуля 00).
- Не меняй Search/Cases/Tickets/Auth, только используй интерфейсы.

Куда класть код (если структура проекта ещё не задана):
- internal/apps/* (service/use-cases)
- internal/apps/http/* (handlers для /api/v1/apps и /settings)
- internal/apps/repository/* (SQL)
Если структура проекта уже есть — следуй ей.

Что нужно сделать (порядок):
1) Реализуй CRUD /api/v1/apps по OpenAPI.
2) Реализуй GET/PUT /api/v1/apps/{id}/settings (atomic replace).
3) Реализуй settings import/export “файлом” (download/upload) по OpenAPI.
4) Реализуй валидацию ключевых настроек:
   - search.default_top_k (min/max)
   - search.confidence_threshold (0..1)
5) Реализуй internal API для Search:
   - получить effective threshold/default_top_k по principal (app_id) + дефолты
6) Покрой unit/integration/contract тестами (включая round-trip export→import).

Definition of Done:
- Search может получить настройки без прямых зависимостей на HTTP.
- Import/export настроек воспроизводим (round-trip).

В ответе (handoff):
- перечисли созданные/изменённые файлы;
- как запустить тесты;
- как реализованы дефолты и валидация.
```

## Цель
Реализовать staff API для управления приложениями и настройками поиска:
- CRUD для `apps`
- чтение/обновление `apps.settings` (JSON)
- импорт/экспорт настроек “файлом”

## Границы модуля
Модуль не реализует поиск напрямую, но должен предоставлять настройки Search-модулю (02) через интерфейс/репозиторий.

## Данные
Таблица `apps`:
- `settings` (JSONB)

Таблица `auth_tokens` (важно для связки app ↔ settings):
- `app_id` для `token_type=app`

## Минимальные настройки (MVP, обязательны)
В `apps.settings` должны поддерживаться ключи:
- `search.default_top_k` (int)
- `search.confidence_threshold` (float)

Допускается расширение настроек без миграций (JSONB).

## API (HTTP)
Следует OpenAPI из модуля 00:
- `GET /api/v1/apps`
- `POST /api/v1/apps`
- `GET /api/v1/apps/{id}`
- `PUT /api/v1/apps/{id}`
- `GET /api/v1/apps/{id}/settings`
- `PUT /api/v1/apps/{id}/settings`
- `GET /api/v1/apps/{id}/settings/export`
- `POST /api/v1/apps/{id}/settings/import`

## Контракт поведения (фиксируем)
- `PUT .../settings` заменяет настройки целиком (atomic replace).
- export возвращает содержимое settings как JSON “файл”.
- import принимает JSON “файл” и заменяет settings целиком.
- Валидация:
  - `default_top_k`: min/max (например 1..50)
  - `confidence_threshold`: диапазон (например 0..1)
  - точные границы фиксируются в OpenAPI.

## Внутренние интерфейсы
- `AppsRepository`:
  - CRUD apps
  - `GetSettings(appID)`
  - `UpdateSettings(appID, settings)`
- `EffectiveSettingsResolver` (опционально):
  - вычислить effective settings по principal (app token → app_id → settings)
  - дефолты, если app_id не найден/настройки пустые

## Тесты
### Unit
- валидация ключей/типов
- дефолты при отсутствии настроек

### Integration
- создать app → обновить settings → экспорт → импорт → совпадает
- связка app token → app_id → Search (02) читает threshold/top_k (в e2e или через интеграционный тест с моками)

### Contract
- соответствие OpenAPI + error schema

## Критерии приёмки
- Search-модуль может стабильно получать `threshold` и `default_top_k` для app токена без “скрытых” зависимостей.

