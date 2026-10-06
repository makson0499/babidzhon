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
	GetAll() ([]models.Expense, error)
	GetByID(id int) (models.Expense, error)
	Update(e models.Expense) error
	Delete(id int) error
}

type Handler struct {
	repo      ExpenseStore
	templates map[string]*template.Template
}

type formData struct {
	Amount, Description, Date string
}

func New(repo ExpenseStore, tmplDir string) (*Handler, error) {
	pages := []string{"expenses.html", "form.html", "about.html"}
	tm := make(map[string]*template.Template)
	layout := filepath.Join(tmplDir, "layout.html")
	for _, p := range pages {
		t, err := template.ParseFiles(layout, filepath.Join(tmplDir, p))
		if err != nil {
			return nil, err
		}
		tm[p] = t
	}
	return &Handler{repo: repo, templates: tm}, nil
}

func (h *Handler) render(w http.ResponseWriter, status int, page string, data map[string]any) {
	t, ok := h.templates[page]
	if !ok {
		http.Error(w, "шаблон не найден", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Println("ошибка шаблона:", err)
	}
}

func serverError(w http.ResponseWriter, err error) {
	log.Println("ошибка:", err)
	http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
}

func parseID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	return id, err == nil && id > 0
}

func parseExpense(r *http.Request) (models.Expense, formData, error) {
	f := formData{
		Amount:      strings.TrimSpace(r.FormValue("amount")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Date:        r.FormValue("date"),
	}
	amount, err := strconv.ParseFloat(strings.Replace(f.Amount, ",", ".", 1), 64)
	if err != nil || amount <= 0 {
		return models.Expense{}, f, errors.New("Сумма должна быть положительным числом.")
	}
	if f.Description == "" {
		return models.Expense{}, f, errors.New("Описание не может быть пустым.")
	}
	date, err := time.Parse("2006-01-02", f.Date)
	if err != nil {
		return models.Expense{}, f, errors.New("Некорректная дата.")
	}
	return models.Expense{Amount: amount, Description: f.Description, Date: date}, f, nil
}


func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

func (h *Handler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.GetAll()
	if err != nil {
		serverError(w, err)
		return
	}
	var total float64
	for _, e := range list {
		total += e.Amount
	}
	h.render(w, http.StatusOK, "expenses.html", map[string]any{
		"Title": "Список трат", "Expenses": list, "Total": total,
	})
}

func (h *Handler) NewExpenseForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "form.html", map[string]any{
		"Title":  "Новая трата",
		"Action": "/expenses",
		"Form":   formData{Date: time.Now().Format("2006-01-02")},
	})
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	e, f, err := parseExpense(r)
	if err != nil {
		h.render(w, http.StatusBadRequest, "form.html", map[string]any{
			"Title": "Новая трата", "Action": "/expenses", "Form": f, "Error": err.Error(),
		})
		return
	}
	if _, err := h.repo.Create(e); err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther) // PRG
}

func (h *Handler) EditExpenseForm(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	e, err := h.repo.GetByID(id)
	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	h.render(w, http.StatusOK, "form.html", map[string]any{
		"Title":  "Редактирование траты",
		"Action": fmt.Sprintf("/expenses/%d/edit", id),
		"Form": formData{
			Amount:      strconv.FormatFloat(e.Amount, 'f', 2, 64),
			Description: e.Description,
			Date:        e.Date.Format("2006-01-02"),
		},
	})
}

func (h *Handler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	e, f, err := parseExpense(r)
	if err != nil {
		h.render(w, http.StatusBadRequest, "form.html", map[string]any{
			"Title": "Редактирование траты", "Action": fmt.Sprintf("/expenses/%d/edit", id),
			"Form": f, "Error": err.Error(),
		})
		return
	}
	e.ID = id
	if err := h.repo.Update(e); errors.Is(err, storage.ErrNotFound) {
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
	if err := h.repo.Delete(id); errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "about.html", map[string]any{"Title": "О проекте"})
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}