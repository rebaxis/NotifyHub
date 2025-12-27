### **Наименование проекта:** Универсальный сервис централизованных оповещений для распределённых агентов
### **Версия ТЗ:** 1.0
### **Язык реализации:** Go (версия ≥ 1.24)

## 1. **Цель проекта**

Разработать **универсальный** сервис оповещений, позволяющий доставлять текстовые уведомления агентам — автономным компонентам (например, `sidecar`-процессам), запущенным в изолированных окружениях.

Сервис реализует паттерн **сервер–агент**, где:

- **Сервер** управляет жизненным циклом оповещений и агентов.
- **Агент** периодически опрашивает сервер и сохраняет полученные оповещения в локальный файл.

Проект **не зависит от типа целевого приложения** (Jupyter, web-сервис и т.д.), а использует абстракции:

- `namespace` — логическая группа агентов (строка, напр. `team-a`, `team-123`, `user-123`)
- `agent_metadata` — произвольные параметры окружения (в т.ч. версии, хэши, теги)

Развёртывание в рамках проекта — **без Kubernetes**: сервер и агенты работают на одном хосте локально.

## 2. **Общие требования**

| Параметр           | Значение                                                                               |
| ------------------ | -------------------------------------------------------------------------------------- |
| Архитектура        | Сервер–агент (pull-based polling)                                                      |
| Язык               | Go                                                                                     |
| Хранение данных    | PostgreSQL ≥ 14                                                                        |
| Аутентификация     | - API-ключ (`agent_secret`) + JWT<br>- токены администраторов (статические)            |
| Формат оповещений  | JSON (сохраняется в файл агентом)                                                      |
| Отказоустойчивость | Сервер и агент должны корректно обрабатывать сетевые сбои, повторять запросы с backoff |
| Логирование        | Structured logging (`log/slog` или `zap`) в `stdout` в формате JSON                    |
| Конфигурация       | Через переменные окружения и/или аргументы командной строки                            |

## 3. **Функциональные требования**

### 3.1. **Сервер оповещений**

#### 3.1.1. Аутентификация

- Все endpoints (кроме `/live`, `/ready`) требуют аутентификации:
    - Для **регистрации агентов** (`/api/v1/agents/register`): заголовок `Authorization: Bearer <AGENT_REGISTRATION_TOKEN>`
    - Для **агентов**: заголовок `X-Agent-ID: <id>` + `X-Agent-Secret: <secret>`  
        _(либо `Authorization: Bearer <secret>`, если `agent_id` в JWT payload`)
    - Для **администраторов**: заголовок `Authorization: Bearer <ADMIN_TOKEN>`

#### 3.1.2. Конфигурация сервера (переменные окружения)

| Переменная                  | Обязательная | Описание                                                    | Пример                                              |
| --------------------------- | ------------ | ----------------------------------------------------------- | --------------------------------------------------- |
| `PORT`                      | ❌            | Порт HTTP-сервера                                           | `8080`                                              |
| `DATABASE_DSN`              | ✅            | Строка подключения к PostgreSQL                             | `postgres://user:pass@localhost:5432/notifyhub`     |
| `ADMIN_TOKEN`               | ✅            | Токен для административных операций                         | `admin-secret-token-123`                            |
| `AGENT_REGISTRATION_TOKEN`  | ✅            | Токен для регистрации новых агентов                         | `registration-token-456`                            |
| `LOG_LEVEL`                 | ❌            | Уровень логирования                                         | `info`, `debug`, `error`                            |

> **Примечание**: Переменные окружения имеют приоритет над флагами командной строки.

#### 3.1.3. Модель данных (PostgreSQL)

##### Таблица `agents`
```
CREATE TABLE agents (
  id             TEXT      PRIMARY KEY, -- предоставленный или сгенерированный agent_id
  namespace      TEXT      NOT NULL, -- логическая группа (аналог k8s namespace)
  metadata       JSONB     NOT NULL DEFAULT '{}', -- произвольные данные: {"jupyter_image_sha":"...", "version":"1.2", "hostname":"..."}
  secret_hash    BYTEA     NOT NULL, -- bcrypt(secret) или SHA256
  last_seen      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  status         TEXT      NOT NULL DEFAULT 'active', -- active, stale, revoked
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

##### Таблица `notifications`
```
CREATE TABLE notifications (
  id                 UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  title              TEXT      NOT NULL,
  body               TEXT      NOT NULL,
  expires_at         TIMESTAMPTZ NOT NULL,
  target_namespaces  TEXT[]    NOT NULL DEFAULT '{}', -- [] — ни для кого; ['*'] — для всех
  target_metadata    JSONB     NOT NULL DEFAULT '{}', -- условия фильтрации по metadata (см. 3.1.4)
  priority           TEXT      NOT NULL DEFAULT 'medium', -- low, medium, high
  created_by         TEXT      NOT NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- индекс для производительности:
CREATE INDEX idx_notif_expires ON notifications (expires_at) WHERE expires_at > NOW();
```

> **Примечание по `target_metadata`**:  
> Поддерживается **частичное совпадение** JSON-объекта.  
> Пример:
> - В `target_metadata = {"jupyter_image_sha":"123", "version":"1"}`
> - Агент с `metadata = {"jupyter_image_sha":"123", "version":"1", "region":"ru"}` → **соответствует**
> - Агент с `metadata = {"jupyter_image_sha":"567", "critical":"true"}` → **не соответствует**

#### 3.1.4. API (HTTP/JSON)

| Метод  | Путь                                      | Описание                                                                                                                                      | Доступ                               |
| ------ | ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| `POST` | `/api/v1/agents/register`                 | Регистрация нового агента:<br>`{ "agent_id": "str", "namespace": "str", "metadata": {} }`<br>→ возвращает `{"secret":"...","agent_id":"..."}` | Токен регистрации                    |
| `PUT`  | `/api/v1/agents/{agent_id}`               | Обновление метаданных и пинг                                                                                                                  | Агент (по JWT)                       |
| `GET`  | `/api/v1/agents/{agent_id}/notifications` | Получить актуальные оповещения для агента                                                                                                     | Агент (по JWT)                       |
| `POST` | `/api/v1/notifications`                   | Создать оповещение (см. структуру ниже)                                                                                                       | Админ (токен)                        |
| `GET`  | `/live`                                   | Liveness probe (`200 OK`)                                                                                                                     | Любой                                |
| `GET`  | `/ready`                                  | Readiness probe (`200` если БД доступна)                                                                                                      | Любой                                |

##### `POST /api/v1/agents/register` — Регистрация агента

**Заголовки:**
```
Authorization: Bearer <AGENT_REGISTRATION_TOKEN>
```

**Запрос:**
```json
{
  "agent_id": "agent-7a3b-4c5d-6e7f",
  "namespace": "team-ml",
  "metadata": {
    "jupyter_image_sha": "sha256:abc123def456",
    "version": "1.4",
    "hostname": "jupyter-server-01",
    "region": "eu-west-1"
  }
}
```

**Ответ (200 OK):**
```json
{
  "agent_id": "agent-7a3b-4c5d-6e7f",
  "secret": "s3cr3t-t0k3n-g3n3r4t3d"
}
```

> **Примечание**: Поле `agent_id` опционально — если не указано, сервер сгенерирует его автоматически.

##### `PUT /api/v1/agents/{agent_id}` — Обновление метаданных агента

**Заголовки:**
```
X-Agent-ID: agent-7a3b-4c5d-6e7f
X-Agent-Secret: s3cr3t-t0k3n-g3n3r4t3d
```

**Запрос:**
```json
{
  "metadata": {
    "jupyter_image_sha": "sha256:abc123def456",
    "version": "1.5",
    "hostname": "jupyter-server-01",
    "region": "eu-west-1",
    "uptime_hours": 24
  }
}
```

**Ответ (200 OK):**
```json
{
  "agent_id": "agent-7a3b-4c5d-6e7f",
  "namespace": "team-ml",
  "metadata": {
    "jupyter_image_sha": "sha256:abc123def456",
    "version": "1.5",
    "hostname": "jupyter-server-01",
    "region": "eu-west-1",
    "uptime_hours": 24
  },
  "status": "active",
  "last_seen": "2025-12-23T14:30:00Z",
  "updated_at": "2025-12-23T14:30:00Z"
}
```

##### `GET /api/v1/agents/{agent_id}/notifications` — Получение оповещений

**Заголовки:**
```
X-Agent-ID: agent-7a3b-4c5d-6e7f
X-Agent-Secret: s3cr3t-t0k3n-g3n3r4t3d
```

**Ответ (200 OK):**
```json
[
  {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "title": "Плановое обслуживание",
    "body": "Сервис будет недоступен с 02:00 до 04:00 20.12.2025",
    "expires_at": "2025-12-20T04:00:00Z",
    "priority": "high",
    "created_by": "Support",
    "created_at": "2025-12-18T10:00:00Z"
  },
  {
    "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
    "title": "Обновление документации",
    "body": "Доступна новая версия API документации",
    "expires_at": "2025-12-25T23:59:59Z",
    "priority": "low",
    "created_by": "DevOps",
    "created_at": "2025-12-19T09:15:00Z"
  }
]
```

##### `POST /api/v1/notifications` — Создание оповещения

**Заголовки:**
```
Authorization: Bearer <ADMIN_TOKEN>
```

**Запрос:**
```json
{
  "title": "Плановое обслуживание",
  "body": "Сервис будет недоступен с 02:00 до 04:00 20.12.2025",
  "expires_at": "2025-12-20T04:00:00Z",
  "target_namespaces": ["team-123", "team-a"],
  "target_metadata": {
    "jupyter_image_sha": "123",
    "version": "8"
  },
  "priority": "high",
  "created_by": "Support"
}
```

**Ответ (201 Created):**
```json
{
  "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "title": "Плановое обслуживание",
  "body": "Сервис будет недоступен с 02:00 до 04:00 20.12.2025",
  "expires_at": "2025-12-20T04:00:00Z",
  "target_namespaces": ["team-123", "team-a"],
  "target_metadata": {
    "jupyter_image_sha": "123",
    "version": "8"
  },
  "priority": "high",
  "created_by": "Support",
  "created_at": "2025-12-18T10:00:00Z"
}
```

> **Примечание по `target_namespaces`**:  
> - `[]` (пустой массив) — оповещение ни для кого  
> - `["*"]` — оповещение для всех агентов  
> - `["team-a", "team-b"]` — только для указанных namespace

##### `GET /live` — Liveness probe

**Ответ (200 OK):**
```json
{
  "status": "ok"
}
```

##### `GET /ready` — Readiness probe

**Ответ (200 OK):**
```json
{
  "status": "ready",
  "database": "connected"
}
```

**Ответ (503 Service Unavailable):**
```json
{
  "status": "not_ready",
  "database": "disconnected",
  "error": "failed to ping database"
}
```

##### Стандартные коды ошибок

**400 Bad Request:**
```json
{
  "error": "validation_error",
  "message": "Invalid request body",
  "details": [
    {
      "field": "namespace",
      "message": "required field is missing"
    }
  ]
}
```

**401 Unauthorized:**
```json
{
  "error": "unauthorized",
  "message": "Invalid or missing authentication credentials"
}
```

**404 Not Found:**
```json
{
  "error": "not_found",
  "message": "Agent with ID 'agent-xyz' not found"
}
```

**500 Internal Server Error:**
```json
{
  "error": "internal_error",
  "message": "An unexpected error occurred",
  "request_id": "req-abc123"
}
```

#### 3.1.5. Логика фильтрации оповещений

При запросе `/agents/{id}/notifications` сервер формирует список, включающий **только** те оповещения, для которых **все** условия выполняются:

1. `NOW() < expires_at`
2. **Одно из**:
    - `target_namespaces` содержит `"*"`
    - `namespace` агента совпадает с одним из элементов `target_namespaces`
3. `agent.metadata` **содержит отличающиеся пары ключ-значение** из `target_metadata` (deep match JSONB != target_metadata)

#### 3.1.6. Фоновые задачи

- Goroutine-воркер, запускаемый при старте сервера:
    - Каждые 5 минут:
        - Удаляет оповещения с `expires_at < NOW() - interval '7 days'`
        - Обновляет `status = 'stale'` для агентов с `last_seen < NOW() - interval '30 minutes'`
    - Логирует статистику: `{"event":"cleanup","deleted_notifications":N,"stale_agents":M}`

### 3.2. **Агент оповещений**

#### 3.2.1. Конфигурация (env-vars)

| Переменная                  | Обязательная | Описание                                                     | Пример                                               |
| --------------------------- | ------------ | ------------------------------------------------------------ | ---------------------------------------------------- |
| `NOTIF_SERVER_URL`          | ✅            | Базовый URL сервера                                          | `http://localhost:8080`                              |
| `AGENT_REGISTRATION_TOKEN`  | ✅            | Токен для регистрации на сервере                             | `registration-token-456`                             |
| `AGENT_NAMESPACE`           | ✅            | Логическая группа агента                                     | `team-ml`, `prod-123`                                |
| `AGENT_METADATA_JSON`       | ✅            | JSON с метаданными                                           | `{"jupyter_image_sha":"sha256:abc","version":"1.4"}` |
| `AGENT_ID`                  | ❌            | Уникальный ID; если не задан — генерируется (`agent-{uuid}`) | `agent-7a3b...`                                      |
| `AGENT_SECRET`              | ❌            | Секрет, используется только при регистрации через /register  | `s3cr3t-t0k3n`                                       |
| `NOTIF_FILE_PATH`           | ❌            | Путь для сохранения оповещений                               | `/opt/app/notifications.json`                        |
| `POLL_INTERVAL_SEC`         | ❌            | Интервал опроса (сек)                                        | `60`                                                 |
| `MAX_RETRY_BACKOFF_SEC`     | ❌            | Макс. задержка при ошибках                                   | `300`                                                |

#### 3.2.2. Алгоритм работы

1. **Инициализация**:
    - Сгенерировать `agent_id`, если не задан.
    - Вызвать `/register` с заголовком `Authorization: Bearer <AGENT_REGISTRATION_TOKEN>`, сохранить полученный `secret` в памяти.
    - Выполнить `PUT /agents/{id}` с метаданными.
2. **Основной цикл** (каждые `POLL_INTERVAL_SEC` сек):
    - Выполнить `PUT /agents/{id}` (обновление `last_seen` + метаданные).
    - Выполнить `GET /agents/{id}/notifications`.
    - При успехе: сохранить тело ответа (массив) в `NOTIF_FILE_PATH` как **валидный JSON**.
    - При ошибке:
        - Логировать ошибку.
        - Ожидать с экспоненциальным backoff (до `MAX_RETRY_BACKOFF_SEC`), затем повторить.
3. **Формат файла оповещений** (`notifications.json`):
```
[
  {
    "id": "a1b2c3...",
    "title": "Плановое обслуживание",
    "body": "Сервис будет недоступен...",
    "expires_at": "2025-12-20T04:00:00Z",
    "priority": "high",
    "received_at": "2025-12-18T14:30:00Z"
  }
]
```
    Поле `received_at` добавляется агентом при сохранении.

#### 3.2.3. Обработка ошибок и отказоустойчивость

- При `401 Unauthorized` (неправильный секрет):
    - Сбросить `secret`, повторить регистрацию.
- При `404 Not Found` для `/agents/{id}`:
    - Считать агента "потерянным", повторить регистрацию.
- При сетевых ошибках (`timeout`, `connection refused`):
    - Повтор с backoff: 5s → 10s → 20s → 40s → ... → `MAX_RETRY_BACKOFF_SEC`.

#### 3.2.4. Graceful shutdown

- Обработка `SIGINT`/`SIGTERM`:
    - Завершить текущий цикл.
    - Выполнить финальный `PUT /agents/{id}` с `status=offline` (опционально).
    - Корректно завершить программу.

## 4. **Нефункциональные требования**

| Категория              | Требование                                                                                                   |
| ---------------------- | ------------------------------------------------------------------------------------------------------------ |
| **Тестируемость**      | Покрытие unit-тестами ≥ 60%                   |

---

## 5. **Состав поставки**

1. Директория `cmd/server/` — main пакет сервера
2. Директория `cmd/agent/` — main пакет агента
3. Директория `doc/` —  документация
3. Директория `internal/` — общая логика:
    - `config/` — загрузка и валидация конфигураций
    - `handler/` — обработчики http запросов
    - `service/` — бизнес логика
    - `repository/` — доступ к данным
    - `db/` — инициализация БД PostgreSQL
    - `model/` — структуры сущностей предметной области (без логики)
    - `lib/` — вспомогательные функции
4. `migrations/` — SQL-миграции
5. `README.md`:
    - Quick start
    - Примеры `curl`

---

## 6. **Используемые библиотеки**

- go.uber.org/fx - Dependency Injection
- github.com/go-chi/chi/v5 - HTTP-роутер
- github.com/go-playground/validator/v10 - валидация данных
- github.com/golang-jwt/jwt/v4
- github.com/jackc/pgx/v5/stdlib
- github.com/golang-migrate/migrate/v4 - SQL-миграции
- github.com/patrickmn/go-cache - кэширование
- go.uber.org/zap - логирование


## 7. **Out of Scope**

- UI для администратора
- RBAC, многоуровневые роли

## 8. **Критерии приёмки**

1. Сервер стартует, создаёт таблицы при первом запуске.
2. Агент регистрируется, получает секрет, сохраняет оповещения в файл.
3. Оповещения фильтруются корректно по `namespace` и `metadata`.
4. `expires_at` учитывается.
5. Агент выдерживает 5-минутный downtime сервера и восстанавливается.
6. Все компоненты собираются через `go build ./...`, проходят `go test ./...`.