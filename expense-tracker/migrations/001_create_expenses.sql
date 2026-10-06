CREATE TABLE IF NOT EXISTS expenses (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    amount      REAL    NOT NULL CHECK (amount > 0),
    description TEXT    NOT NULL,
    date        TEXT    NOT NULL          -- формат YYYY-MM-DD
);

CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);