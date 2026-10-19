CREATE TABLE IF NOT EXISTS categories (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT    NOT NULL UNIQUE
);

INSERT OR IGNORE INTO categories (name) VALUES
    ('Еда'),
    ('Транспорт'),
    ('Жильё'),
    ('Развлечения'),
    ('Здоровье'),
    ('Одежда'),
    ('Другое');

-- внешний ключ: category_id ссылается на categories.id
ALTER TABLE expenses ADD COLUMN category_id INTEGER REFERENCES categories(id);

-- старым тратам (из этапа 3) ставим категорию «Другое»
UPDATE expenses
SET category_id = (SELECT id FROM categories WHERE name = 'Другое')
WHERE category_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses(category_id);