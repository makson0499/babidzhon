package models

import "time"

type Category struct {
	ID   int
	Name string
}

type Expense struct {
	ID           int
	Amount       float64
	Description  string
	Date         time.Time
	CategoryID   int
	CategoryName string // заполняется через JOIN, в таблице expenses его нет
}

// ExpenseFilter — необязательные фильтры списка. Нулевое значение = «фильтр не задан».
type ExpenseFilter struct {
	CategoryID int    // 0 — все категории
	From       string // "YYYY-MM-DD" или ""
	To         string // "YYYY-MM-DD" или ""
}