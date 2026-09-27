package main

import (
	"log"
	"net/http"

	"expense-tracker/internal/handlers"
	"expense-tracker/internal/storage"
)

func main() {
	store := storage.NewMemoryStore()

	h, err := handlers.New(store, "web/templates")
	if err != nil {
		log.Fatal("не удалось загрузить шаблоны: ", err)
	}

	mux := http.NewServeMux()

	// статика: /static/style.css → web/static/style.css
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("GET /{$}", h.Home) // {$} — только точный путь "/"
	mux.HandleFunc("GET /expenses", h.ListExpenses)
	mux.HandleFunc("GET /expenses/new", h.NewExpenseForm)
	mux.HandleFunc("POST /expenses", h.CreateExpense)
	mux.HandleFunc("GET /about", h.About)
	mux.HandleFunc("GET /ping", h.Ping) // при другом методе ServeMux сам вернёт 405

	addr := ":8080"
	log.Println("Сервер запущен на http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
