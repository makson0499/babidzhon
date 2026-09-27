package models

import "time"

// Expense — одна трата.
type Expense struct {
	ID          int
	Amount      float64
	Description string
	Date        time.Time
}
