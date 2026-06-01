# Модуль 05 — Apps & Settings (JSON settings + import/export)

## Промпт для изолированного агента (копипаст)

```text
Ты изолированный агент-разработчик. Твоя задача — реализовать Модуль 05 (Apps & Settings): CRUD приложений и управление settings (JSON) с импортом/экспортом.

Контекст:
- Основа требований: docs/main_doc и docs/agents/README.md, а также текущий файл docs/agents/05-apps-settings.md.
- Контракты и миграции задаёт Модуль 00 (contracts/openapi.yaml и db/migrations/*). Считай их “замороженными”.
- Search (02) будет читать threshold/default_top_k отсюда.
- Auth (01) даёт principal и для app токенов — app_id, по которому нужно найти settings.
- Ключевые решения: дефолты только в 05; интерфейс AppSettingsReader в search, реализация в 05; роутинг в Glue; import = JSON body как PUT settings; 409 при дубликате имени, 422 при валидации; все эндпоинты apps — только staff. Подробно — секция «Уточнения».

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

## Уточнения (решения по открытым вопросам)

1. **Дефолты для staff / отсутствующего app**  
   **Единственный источник дефолтов — модуль 05.** Константы (например threshold=0.7, defaultTopK=10) задаются в 05 (константы в коде или позже конфиг/переменные окружения). Search (02) не хранит свои дефолты; получает значения только через интерфейс AppSettingsReader. Для staff-токена или когда app_id нет/не найден реализация в 05 возвращает эти же константы. Числа можно совпадать с теми, что использовались в 02 как временный fallback, но формально дефолты живут только в 05.

2. **Где объявлять интерфейс для Search**  
   **Интерфейс AppSettingsReader остаётся в internal/search** (interfaces.go). Реализацию из 05 (например internal/apps/effectivesettings.go или аналог) подключает Glue и передаёт в Search. Интерфейс в internal/apps не переносим.

3. **Роутинг /api/v1/apps**  
   В 05 только **handler’ы** (как у cases/tickets: разбор path/method в ServeHTTP или аналог). Регистрацию маршрутов в cmd/ или общий роутер делает Glue (08). В рамках 05 не добавляем вызовы Handle/HandleFunc в общий роутер — только код, который Glue потом подключит.

4. **Import: только JSON body**  
   По OpenAPI import — requestBody application/json (схема AppSettings). **Import = тот же контракт, что и PUT .../settings:** тело JSON, полная замена настроек. Отдельного multipart/file upload нет; при необходимости «файл» — это просто JSON в теле запроса.

5. **409 при создании app**  
   **409** — при нарушении уникальности имени (констрейнт `uq_apps_name`). **422** — при ошибках валидации тела (пустой name, несоответствие схеме и т.п.). Так и зафиксировать в реализации и тестах.

6. **Эндпоинты apps: только staff**  
   Все операции `GET/POST/PUT /api/v1/apps`, `.../settings`, `.../export`, `.../import` доступны **только со staff-токеном**. Search — app+staff; остальное staff. Реализация: перед handler’ами apps в Glue вешается RequireStaff (модуль 01).

## Внутренние интерфейсы
- `AppsRepository`:
  - CRUD apps
  - `GetSettings(appID)`
  - `UpdateSettings(appID, settings)`
- Реализация **AppSettingsReader** (интерфейс в internal/search): по principal (app_id или staff) возвращает effective threshold и defaultTopK; при отсутствии app/настроек — дефолты из 05.

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

