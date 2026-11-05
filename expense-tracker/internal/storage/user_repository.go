package storage

import (
	"database/sql"
	"errors"
	"strings"

	"expense-tracker/internal/models"
)

var ErrEmailTaken = errors.New("email уже зарегистрирован")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create сохраняет пользователя. Уникальность email проверяет сама база (UNIQUE).
func (r *UserRepository) Create(email, passwordHash string) (int, error) {
	res, err := r.db.Exec(
		`INSERT INTO users (email, password_hash) VALUES (?, ?)`, email, passwordHash,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, ErrEmailTaken
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *UserRepository) GetByEmail(email string) (models.User, error) {
	var u models.User
	err := r.db.QueryRow(
		`SELECT id, email, password_hash FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}