package models

import "time"

type Category struct {
	ID   int
	Name string
}

type Expense struct {
	ID           int
	UserID       int
	Amount       float64
	Description  string
	Date         time.Time
	CategoryID   int
	CategoryName string
}

// ExpenseFilter: UserID обязателен, остальные поля — необязательные фильтры.
type ExpenseFilter struct {
	UserID     int
	CategoryID int
	From       string
	To         string
}

// HasFilters сообщает, выбрал ли пользователь хоть один фильтр (UserID не считается).
func (f ExpenseFilter) HasFilters() bool {
	return f.CategoryID > 0 || f.From != "" || f.To != ""
}