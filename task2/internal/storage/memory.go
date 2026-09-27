package storage

import (
	"sort"
	"sync"

	"expense-tracker/internal/models"
)

type MemoryStore struct {
	mu       sync.RWMutex
	expenses []models.Expense
	nextID   int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1}
}

func (s *MemoryStore) Add(e models.Expense) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e.ID = s.nextID
	s.nextID++
	s.expenses = append(s.expenses, e)
}

// All возвращает копию списка: сначала новые траты.
func (s *MemoryStore) All() []models.Expense {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Expense, len(s.expenses))
	copy(out, s.expenses)
	sort.Slice(out, func(i, j int) bool { return out[i].Date.After(out[j].Date) })
	return out
}
