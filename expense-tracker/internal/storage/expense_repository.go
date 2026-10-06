package storage

import (
	"database/sql"
	"errors"
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

func (r *ExpenseRepository) Create(e models.Expense) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO expenses (amount, description, date) VALUES (?, ?, ?)`,
		e.Amount, e.Description, e.Date.Format(dateLayout),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *ExpenseRepository) GetAll() ([]models.Expense, error) {
	rows, err := r.db.Query(
		`SELECT id, amount, description, date FROM expenses ORDER BY date DESC, id DESC`,
	)
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
	row := r.db.QueryRow(
		`SELECT id, amount, description, date FROM expenses WHERE id = ?`, id,
	)
	e, err := scanExpense(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Expense{}, ErrNotFound
	}
	return e, err
}

func (r *ExpenseRepository) Update(e models.Expense) error {
	res, err := r.db.Exec(
		`UPDATE expenses SET amount = ?, description = ?, date = ? WHERE id = ?`,
		e.Amount, e.Description, e.Date.Format(dateLayout), e.ID,
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
	if err := s.Scan(&e.ID, &e.Amount, &e.Description, &date); err != nil {
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