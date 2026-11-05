package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"expense-tracker/internal/models"
	"expense-tracker/internal/storage"
)

type ExpenseStore interface {
	Create(e models.Expense) (int, error)
	GetAll(f models.ExpenseFilter) ([]models.Expense, error)
	GetByID(userID, id int) (models.Expense, error)
	Update(e models.Expense) error
	Delete(userID, id int) error
}

type CategoryStore interface {
	GetAll() ([]models.Category, error)
	Exists(id int) (bool, error)
}

type UserStore interface {
	Create(email, passwordHash string) (int, error)
	GetByEmail(email string) (models.User, error)
}

type SessionStore interface {
	Create(userID int, ttl time.Duration) (string, time.Time, error)
	GetUser(token string) (models.User, error)
	Delete(token string) error
}

type Handler struct {
	expenses   ExpenseStore
	categories CategoryStore
	users      UserStore
	sessions   SessionStore
	templates  map[string]*template.Template
}

type formData struct {
	Amount, Description, Date string
	CategoryID                int
}

const dateLayout = "2006-01-02"

func New(expenses ExpenseStore, categories CategoryStore, users UserStore,
	sessions SessionStore, tmplDir string) (*Handler, error) {

	pages := []string{"expenses.html", "form.html", "about.html", "login.html", "register.html"}
	tm := make(map[string]*template.Template)
	layout := filepath.Join(tmplDir, "layout.html")
	for _, p := range pages {
		t, err := template.ParseFiles(layout, filepath.Join(tmplDir, p))
		if err != nil {
			return nil, err
		}
		tm[p] = t
	}
	return &Handler{
		expenses: expenses, categories: categories,
		users: users, sessions: sessions, templates: tm,
	}, nil
}

// render добавляет в данные шаблона текущего пользователя (для шапки сайта).
func (h *Handler) render(w http.ResponseWriter, r *http.Request, status int, page string, data map[string]any) {
	t, ok := h.templates[page]
	if !ok {
		http.Error(w, "шаблон не найден", http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	if u, ok := UserFromContext(r.Context()); ok {
		data["User"] = u
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Println("ошибка шаблона:", err)
	}
}

func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, status int,
	title, action string, f formData, errMsg string) {

	cats, err := h.categories.GetAll()
	if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, r, status, "form.html", map[string]any{
		"Title": title, "Action": action, "Form": f, "Error": errMsg, "Categories": cats,
	})
}

func serverError(w http.ResponseWriter, err error) {
	log.Println("ошибка:", err)
	http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
}

func parseID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	return id, err == nil && id > 0
}

func (h *Handler) parseExpense(r *http.Request) (models.Expense, formData, error) {
	catID, _ := strconv.Atoi(r.FormValue("category_id"))
	f := formData{
		Amount:      strings.TrimSpace(r.FormValue("amount")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Date:        r.FormValue("date"),
		CategoryID:  catID,
	}
	amount, err := strconv.ParseFloat(strings.Replace(f.Amount, ",", ".", 1), 64)
	if err != nil || amount <= 0 {
		return models.Expense{}, f, errors.New("Сумма должна быть положительным числом.")
	}
	if f.Description == "" {
		return models.Expense{}, f, errors.New("Описание не может быть пустым.")
	}
	date, err := time.Parse(dateLayout, f.Date)
	if err != nil {
		return models.Expense{}, f, errors.New("Некорректная дата.")
	}
	if catID <= 0 {
		return models.Expense{}, f, errors.New("Выберите категорию.")
	}
	ok, err := h.categories.Exists(catID)
	if err != nil {
		return models.Expense{}, f, err
	}
	if !ok {
		return models.Expense{}, f, errors.New("Такой категории не существует.")
	}
	return models.Expense{
		UserID:      currentUserID(r),
		Amount:      amount,
		Description: f.Description,
		Date:        date,
		CategoryID:  catID,
	}, f, nil
}

func parseFilter(r *http.Request) (models.ExpenseFilter, string) {
	q := r.URL.Query()
	f := models.ExpenseFilter{UserID: currentUserID(r)}
	var problems []string

	if s := q.Get("category"); s != "" {
		if id, err := strconv.Atoi(s); err == nil && id > 0 {
			f.CategoryID = id
		} else {
			problems = append(problems, "некорректная категория")
		}
	}
	if s := q.Get("from"); s != "" {
		if _, err := time.Parse(dateLayout, s); err == nil {
			f.From = s
		} else {
			problems = append(problems, "некорректная дата «с»")
		}
	}
	if s := q.Get("to"); s != "" {
		if _, err := time.Parse(dateLayout, s); err == nil {
			f.To = s
		} else {
			problems = append(problems, "некорректная дата «по»")
		}
	}
	if f.From != "" && f.To != "" && f.From > f.To {
		problems = append(problems, "дата «с» позже даты «по»")
	}

	msg := ""
	if len(problems) > 0 {
		msg = "Фильтр: " + strings.Join(problems, ", ") + "."
	}
	return f, msg
}

// ---------- маршруты ----------

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

func (h *Handler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	filter, filterErr := parseFilter(r)

	list, err := h.expenses.GetAll(filter)
	if err != nil {
		serverError(w, err)
		return
	}
	cats, err := h.categories.GetAll()
	if err != nil {
		serverError(w, err)
		return
	}

	var total float64
	for _, e := range list {
		total += e.Amount
	}
	h.render(w, r, http.StatusOK, "expenses.html", map[string]any{
		"Title":       "Список трат",
		"Expenses":    list,
		"Total":       total,
		"Categories":  cats,
		"Filter":      filter,
		"Filtered":    filter.HasFilters(),
		"FilterError": filterErr,
	})
}

func (h *Handler) NewExpenseForm(w http.ResponseWriter, r *http.Request) {
	h.renderForm(w, r, http.StatusOK, "Новая трата", "/expenses",
		formData{Date: time.Now().Format(dateLayout)}, "")
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	e, f, err := h.parseExpense(r)
	if err != nil {
		h.renderForm(w, r, http.StatusBadRequest, "Новая трата", "/expenses", f, err.Error())
		return
	}
	if _, err := h.expenses.Create(e); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) EditExpenseForm(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	e, err := h.expenses.GetByID(currentUserID(r), id)
	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r) // чужая трата выглядит как несуществующая
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	h.renderForm(w, r, http.StatusOK, "Редактирование траты", fmt.Sprintf("/expenses/%d/edit", id),
		formData{
			Amount:      strconv.FormatFloat(e.Amount, 'f', 2, 64),
			Description: e.Description,
			Date:        e.Date.Format(dateLayout),
			CategoryID:  e.CategoryID,
		}, "")
}

func (h *Handler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	action := fmt.Sprintf("/expenses/%d/edit", id)
	e, f, err := h.parseExpense(r)
	if err != nil {
		h.renderForm(w, r, http.StatusBadRequest, "Редактирование траты", action, f, err.Error())
		return
	}
	e.ID = id
	if err := h.expenses.Update(e); errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.expenses.Delete(currentUserID(r), id); errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, "about.html", map[string]any{"Title": "О проекте"})
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}