package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"expense-tracker/internal/storage"
)

const sessionTTL = 7 * 24 * time.Hour // сессия живёт неделю

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,                 // JavaScript на странице не может прочитать cookie
		SameSite: http.SameSiteLaxMode, // cookie не отправляется в POST с чужих сайтов
		// Secure: true — включить, когда сайт работает по HTTPS
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
}

func (h *Handler) startSession(w http.ResponseWriter, userID int) error {
	token, expires, err := h.sessions.Create(userID, sessionTTL)
	if err != nil {
		return err
	}
	setSessionCookie(w, token, expires)
	return nil
}

func validEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

// ---------- регистрация ----------

// GET /register
func (h *Handler) RegisterForm(w http.ResponseWriter, r *http.Request) {
	if _, ok := UserFromContext(r.Context()); ok {
		http.Redirect(w, r, "/expenses", http.StatusSeeOther)
		return
	}
	h.render(w, r, http.StatusOK, "register.html", map[string]any{"Title": "Регистрация"})
}

// POST /register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	fail := func(msg string) {
		h.render(w, r, http.StatusBadRequest, "register.html", map[string]any{
			"Title": "Регистрация", "Email": email, "Error": msg,
		})
	}

	if !validEmail(email) {
		fail("Введите корректный email.")
		return
	}
	// bcrypt учитывает только первые 72 байта пароля
	if len(password) < 8 || len(password) > 72 {
		fail("Пароль должен быть от 8 до 72 символов.")
		return
	}
	if password != confirm {
		fail("Пароли не совпадают.")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		serverError(w, err)
		return
	}

	userID, err := h.users.Create(email, string(hash))
	if errors.Is(err, storage.ErrEmailTaken) {
		fail("Пользователь с таким email уже зарегистрирован.")
		return
	} else if err != nil {
		serverError(w, err)
		return
	}

	// сразу после регистрации пользователь становится вошедшим
	if err := h.startSession(w, userID); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

// ---------- вход ----------

// GET /login
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if _, ok := UserFromContext(r.Context()); ok {
		http.Redirect(w, r, "/expenses", http.StatusSeeOther)
		return
	}
	h.render(w, r, http.StatusOK, "login.html", map[string]any{"Title": "Вход"})
}

// POST /login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")

	// Одинаковое сообщение в обоих случаях: не подсказываем, какие email зарегистрированы.
	fail := func() {
		h.render(w, r, http.StatusUnauthorized, "login.html", map[string]any{
			"Title": "Вход", "Email": email, "Error": "Неверный email или пароль.",
		})
	}

	u, err := h.users.GetByEmail(email)
	if errors.Is(err, storage.ErrNotFound) {
		fail()
		return
	} else if err != nil {
		serverError(w, err)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		fail()
		return
	}

	// если в браузере осталась старая сессия, удаляем её
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = h.sessions.Delete(c.Value)
	}
	if err := h.startSession(w, u.ID); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

// ---------- выход ----------

// POST /logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		// удаляем сессию на сервере: даже скопированная cookie перестанет работать
		if err := h.sessions.Delete(c.Value); err != nil {
			log.Println("ошибка удаления сессии:", err)
		}
	}
	clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}