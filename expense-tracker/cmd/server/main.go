package main

import (
	"log"
	"net/http"

	"expense-tracker/internal/handlers"
	"expense-tracker/internal/storage"
)

func main() {
	db, err := storage.Open("expenses.db") // DSN — путь к файлу (на этапе 8 уйдёт в .env)
	if err != nil {
		log.Fatal("не удалось открыть БД: ", err)
	}
	defer db.Close()

	if err := storage.Migrate(db, "migrations"); err != nil {
		log.Fatal(err)
	}

	repo := storage.NewExpenseRepository(db)

	h, err := handlers.New(repo, "web/templates")
	if err != nil {
		log.Fatal("не удалось загрузить шаблоны: ", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /expenses", h.ListExpenses)
	mux.HandleFunc("GET /expenses/new", h.NewExpenseForm)
	mux.HandleFunc("POST /expenses", h.CreateExpense)
	mux.HandleFunc("GET /expenses/{id}/edit", h.EditExpenseForm)
	mux.HandleFunc("POST /expenses/{id}/edit", h.UpdateExpense)
	mux.HandleFunc("POST /expenses/{id}/delete", h.DeleteExpense)
	mux.HandleFunc("GET /about", h.About)
	mux.HandleFunc("GET /ping", h.Ping)

	log.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}