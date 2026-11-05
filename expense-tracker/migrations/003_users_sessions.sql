CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,              -- только хэш
    created_at    TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT    PRIMARY KEY,              -- случайный токен из cookie
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL                  -- Unix-время окончания сессии
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

-- каждая трата принадлежит пользователю
ALTER TABLE expenses ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_expenses_user ON expenses(user_id);