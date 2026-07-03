# TG Dnevnik Bot — Design Spec

## Overview
Telegram-бот для ведения личного дневника. Все действия через inline-кнопки,
минимум текстовых команд (только `/start`).

## Tech Stack
- **Language:** Go
- **TG Library:** `tucnak/telebot` v3
- **Database:** PostgreSQL
- **DB access:** `pgx` (raw SQL)
- **Migrations:** `golang-migrate/migrate`
- **Infra:** Docker + docker-compose

## Project Structure
```
bot/
├── cmd/bot/main.go
├── internal/
│   ├── handler/       — TG handlers (commands, callbacks, states)
│   ├── service/       — business logic
│   ├── repository/    — DB queries
│   └── model/         — data structures
├── migrations/
│   ├── 000001_create_entries.up.sql
│   └── 000001_create_entries.down.sql
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── go.sum
```

## Database

### Table `entries`
| Column      | Type                    | Notes              |
|-------------|-------------------------|--------------------|
| id          | SERIAL PRIMARY KEY      | Auto-increment     |
| user_id     | BIGINT NOT NULL          | TG user ID         |
| content     | TEXT NOT NULL            | Entry text         |
| created_at  | TIMESTAMPTZ DEFAULT NOW  | Immutable          |
| updated_at  | TIMESTAMPTZ DEFAULT NOW  | Updated on edit    |

Index: `CREATE INDEX idx_entries_user_created ON entries(user_id, created_at DESC);`

## Bot Commands & Flows

### `/start`
- Приветствие + 3 кнопки:
  - 📝 Новая запись
  - 📋 Мои записи
  - 🎲 Сюрприз (random entry)

### Flow: Новая запись
1. Нажатие 📝 → бот: "✍️ Напиши свою запись:" + кнопка ❌ Отмена
2. Пользователь вводит текст (или /cancel)
3. Сохранение → "✅ Готово!" + кнопка 📋 К списку

### Flow: Список записей (пагинация)
1. Показывается 5 записей на странице
2. Каждая запись: `📅 03.07.2026 — первые 50 символов...`
3. Под каждой — inline-кнопки: [👁] [✏️] [🗑]
4. Навигация снизу: ◀️ 1/4 ▶️

### Flow: Просмотр
- Полный текст + дата создания/обновления
- Кнопки: [✏️ Редактировать] [🗑 Удалить] [⬅️ Назад]

### Flow: Редактирование
1. ✏️ → "✍️ Введи новый текст:" (или показываем текущий)
2. Пользователь присылает новый текст
3. "✅ Обновлено!"

### Flow: Удаление
- 🗑 → подтверждение: "Точно удалить?" [✅ Да] [❌ Нет]
- Да → "🗑 Удалено"

## State Management
In-memory map `map[int64]*UserState` с sync.RWMutex.
Ключ — TG userID, значение — текущее состояние (idle, creating, editing + editEntryID).
Middleware-хендлер считывает/записывает состояние на каждый апдейт.

## Docker
- `docker-compose.yml`:
  - `db` — PostgreSQL 16, volume для данных, healthcheck
  - `migrations` — образ `golang-migrate/migrate`, depends_on db (healthy), накатывает .up.sql
  - `bot` — собирается из Dockerfile, depends_on migrations
- `Dockerfile`: multi-stage build (Go build → distroless)

## Interactive Features (Brainstorming — TBD)
- 🎲 Сюрприз — случайная запись из дневника
- Можно добавить позже: /mood, /stats, /streak, /search
