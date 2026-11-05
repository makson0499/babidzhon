package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"expense-tracker/internal/models"
	"expense-tracker/internal/storage"
)

const sessionCookie = "session_id"

// Неэкспортируемый тип ключа: другой пакет не сможет случайно
// перезаписать наше значение в контексте своим ключом.
type contextKey struct{}

var userKey = contextKey{}

func withUser(ctx context.Context, u models.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext достаёт текущего пользователя, положенного туда middleware.
func UserFromContext(ctx context.Context) (models.User, bool) {
	u, ok := ctx.Value(userKey).(models.User)
	return u, ok
}

// currentUserID используется в обработчиках за RequireAuth, где пользователь гарантированно есть.
func currentUserID(r *http.Request) int {
	u, _ := UserFromContext(r.Context())
	return u.ID
}

// LoadUser: cookie → сессия в БД → пользователь в context.
func (h *Handler) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err == nil && c.Value != "" {
			u, err := h.sessions.GetUser(c.Value)
			switch {
			case err == nil:
				r = r.WithContext(withUser(r.Context(), u))
			case errors.Is(err, storage.ErrNotFound):
				clearSessionCookie(w) // сессия истекла или удалена, cookie больше не нужна
			default:
				log.Println("ошибка проверки сессии:", err)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth пропускает запрос дальше только для вошедшего пользователя.
func (h *Handler) RequireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	})
}