package storage

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"expense-tracker/internal/models"
)

const dateLayout = "2006-01-02"

var ErrNotFound = errors.New("запись не найдена")

type ExpenseRepository struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

// Общая часть SELECT: трата + название категории одним запросом (JOIN).
// LEFT JOIN — чтобы трата без категории тоже попала в результат.
const selectExpenses = `
	SELECT e.id, e.amount, e.description, e.date,
	       COALESCE(e.category_id, 0),
	       COALESCE(c.name, 'Без категории')
	FROM expenses e
	LEFT JOIN categories c ON c.id = e.category_id`

func (r *ExpenseRepository) Create(e models.Expense) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO expenses (amount, description, date, category_id) VALUES (?, ?, ?, ?)`,
		e.Amount, e.Description, e.Date.Format(dateLayout), e.CategoryID,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// GetAll возвращает траты с учётом фильтра. Пустой фильтр = все траты.
func (r *ExpenseRepository) GetAll(f models.ExpenseFilter) ([]models.Expense, error) {
	var conds []string // фрагменты WHERE — только константные строки из кода
	var args []any     // значения — только через placeholder ?

	if f.CategoryID > 0 {
		conds = append(conds, "e.category_id = ?")
		args = append(args, f.CategoryID)
	}
	if f.From != "" {
		conds = append(conds, "e.date >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		conds = append(conds, "e.date <= ?")
		args = append(args, f.To)
	}

	query := selectExpenses
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY e.date DESC, e.id DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Expense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (r *ExpenseRepository) GetByID(id int) (models.Expense, error) {
	row := r.db.QueryRow(selectExpenses+" WHERE e.id = ?", id)
	e, err := scanExpense(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Expense{}, ErrNotFound
	}
	return e, err
}

func (r *ExpenseRepository) Update(e models.Expense) error {
	res, err := r.db.Exec(
		`UPDATE expenses SET amount = ?, description = ?, date = ?, category_id = ? WHERE id = ?`,
		e.Amount, e.Description, e.Date.Format(dateLayout), e.CategoryID, e.ID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *ExpenseRepository) Delete(id int) error {
	res, err := r.db.Exec(`DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanExpense(s scanner) (models.Expense, error) {
	var e models.Expense
	var date string
	if err := s.Scan(&e.ID, &e.Amount, &e.Description, &date, &e.CategoryID, &e.CategoryName); err != nil {
		return e, err
	}
	t, err := time.Parse(dateLayout, date)
	if err != nil {
		return e, err
	}
	e.Date = t
	return e, nil
}

func checkAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}