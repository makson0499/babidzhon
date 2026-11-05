package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"expense-tracker/internal/models"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create создаёт сессию со случайным токеном (32 байта из криптостойкого генератора).
func (r *SessionRepository) Create(userID int, ttl time.Duration) (string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	expires := time.Now().Add(ttl)

	_, err := r.db.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expires.Unix(),
	)
	return token, expires, err
}

// GetUser возвращает владельца сессии, но только если срок сессии не истёк.
func (r *SessionRepository) GetUser(token string) (models.User, error) {
	var u models.User
	err := r.db.QueryRow(`
		SELECT u.id, u.email
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > ?`,
		token, time.Now().Unix(),
	).Scan(&u.ID, &u.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (r *SessionRepository) Delete(token string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE id = ?`, token)
	return err
}

// DeleteExpired чистит таблицу от просроченных сессий.
func (r *SessionRepository) DeleteExpired() (int64, error) {
	res, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}