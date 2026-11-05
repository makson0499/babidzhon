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

const selectExpenses = `
	SELECT e.id, e.user_id, e.amount, e.description, e.date,
	       COALESCE(e.category_id, 0),
	       COALESCE(c.name, 'Без категории')
	FROM expenses e
	LEFT JOIN categories c ON c.id = e.category_id`

func (r *ExpenseRepository) Create(e models.Expense) (int, error) {
	if e.UserID <= 0 {
		return 0, errors.New("не указан владелец траты")
	}
	res, err := r.db.Exec(
		`INSERT INTO expenses (user_id, amount, description, date, category_id) VALUES (?, ?, ?, ?, ?)`,
		e.UserID, e.Amount, e.Description, e.Date.Format(dateLayout), e.CategoryID,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *ExpenseRepository) GetAll(f models.ExpenseFilter) ([]models.Expense, error) {
	if f.UserID <= 0 {
		return nil, errors.New("не указан пользователь")
	}

	// условие по пользователю есть всегда
	conds := []string{"e.user_id = ?"}
	args := []any{f.UserID}

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

	query := selectExpenses + " WHERE " + strings.Join(conds, " AND ") +
		" ORDER BY e.date DESC, e.id DESC"

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

// GetByID находит трату только среди трат указанного пользователя.
func (r *ExpenseRepository) GetByID(userID, id int) (models.Expense, error) {
	row := r.db.QueryRow(selectExpenses+" WHERE e.id = ? AND e.user_id = ?", id, userID)
	e, err := scanExpense(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Expense{}, ErrNotFound
	}
	return e, err
}

func (r *ExpenseRepository) Update(e models.Expense) error {
	res, err := r.db.Exec(
		`UPDATE expenses SET amount = ?, description = ?, date = ?, category_id = ?
		 WHERE id = ? AND user_id = ?`,
		e.Amount, e.Description, e.Date.Format(dateLayout), e.CategoryID, e.ID, e.UserID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

func (r *ExpenseRepository) Delete(userID, id int) error {
	res, err := r.db.Exec(`DELETE FROM expenses WHERE id = ? AND user_id = ?`, id, userID)
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
	if err := s.Scan(&e.ID, &e.UserID, &e.Amount, &e.Description, &date,
		&e.CategoryID, &e.CategoryName); err != nil {
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