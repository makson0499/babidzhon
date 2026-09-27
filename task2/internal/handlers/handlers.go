package handlers

import (
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

type Handler struct {
	store     *storage.MemoryStore
	templates map[string]*template.Template
}

func New(store *storage.MemoryStore, tmplDir string) (*Handler, error) {
	pages := []string{"expenses.html", "new.html", "about.html"}
	tm := make(map[string]*template.Template)
	layout := filepath.Join(tmplDir, "layout.html")
	for _, p := range pages {
		t, err := template.ParseFiles(layout, filepath.Join(tmplDir, p))
		if err != nil {
			return nil, err
		}
		tm[p] = t
	}
	return &Handler{store: store, templates: tm}, nil
}

func (h *Handler) render(w http.ResponseWriter, page string, data map[string]any) {
	t, ok := h.templates[page]
	if !ok {
		http.Error(w, "шаблон не найден", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Println("ошибка шаблона:", err)
		http.Error(w, "ошибка рендеринга", http.StatusInternalServerError)
	}
}

// GET / → перенаправляем на список
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/expenses", http.StatusFound)
}

// GET /expenses
func (h *Handler) ListExpenses(w http.ResponseWriter, r *http.Request) {
	list := h.store.All()
	var total float64
	for _, e := range list {
		total += e.Amount
	}
	h.render(w, "expenses.html", map[string]any{
		"Title":    "Список трат",
		"Expenses": list,
		"Total":    total,
	})
}

// GET /expenses/new
func (h *Handler) NewExpenseForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, "new.html", map[string]any{
		"Title": "Новая трата",
		"Today": time.Now().Format("2006-01-02"),
	})
}

// POST /expenses
func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "некорректная форма", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(strings.Replace(r.FormValue("amount"), ",", ".", 1), 64)
	desc := strings.TrimSpace(r.FormValue("description"))
	date, dateErr := time.Parse("2006-01-02", r.FormValue("date"))

	if err != nil || amount <= 0 || desc == "" || dateErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "new.html", map[string]any{
			"Title": "Новая трата",
			"Today": time.Now().Format("2006-01-02"),
			"Error": "Проверьте поля: сумма > 0, описание не пустое, дата корректна.",
		})
		return
	}

	h.store.Add(models.Expense{Amount: amount, Description: desc, Date: date})

	// Post/Redirect/Get: после POST страницу не рендерим, а делаем перенаправление
	http.Redirect(w, r, "/expenses", http.StatusSeeOther) // 303
}

// GET /about
func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	h.render(w, "about.html", map[string]any{"Title": "О проекте"})
}

// GET /ping (из этапа 1)
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}
