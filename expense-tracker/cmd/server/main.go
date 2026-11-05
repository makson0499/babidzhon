package main

import (
	"log"
	"net/http"
	"time"

	"expense-tracker/internal/handlers"
	"expense-tracker/internal/storage"
)

func main() {
	db, err := storage.Open("expenses.db?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatal("не удалось открыть БД: ", err)
	}
	defer db.Close()

	if err := storage.Migrate(db, "migrations"); err != nil {
		log.Fatal(err)
	}

	expenseRepo := storage.NewExpenseRepository(db)
	categoryRepo := storage.NewCategoryRepository(db)
	userRepo := storage.NewUserRepository(db)
	sessionRepo := storage.NewSessionRepository(db)

	// фоновая очистка просроченных сессий: сразу при старте и потом раз в час
	go func() {
		for {
			if n, err := sessionRepo.DeleteExpired(); err != nil {
				log.Println("очистка сессий:", err)
			} else if n > 0 {
				log.Printf("удалено просроченных сессий: %d", n)
			}
			time.Sleep(time.Hour)
		}
	}()

	h, err := handlers.New(expenseRepo, categoryRepo, userRepo, sessionRepo, "web/templates")
	if err != nil {
		log.Fatal("не удалось загрузить шаблоны: ", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// открытые маршруты
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /about", h.About)
	mux.HandleFunc("GET /ping", h.Ping)
	mux.HandleFunc("GET /register", h.RegisterForm)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("GET /login", h.LoginForm)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /logout", h.Logout)

	// закрытые маршруты: только для вошедших
	mux.Handle("GET /expenses", h.RequireAuth(h.ListExpenses))
	mux.Handle("GET /expenses/new", h.RequireAuth(h.NewExpenseForm))
	mux.Handle("POST /expenses", h.RequireAuth(h.CreateExpense))
	mux.Handle("GET /expenses/{id}/edit", h.RequireAuth(h.EditExpenseForm))
	mux.Handle("POST /expenses/{id}/edit", h.RequireAuth(h.UpdateExpense))
	mux.Handle("POST /expenses/{id}/delete", h.RequireAuth(h.DeleteExpense))

	// цепочка: запрос → LoadUser → mux (маршрутизация) → RequireAuth → обработчик
	handler := h.LoadUser(mux)

	log.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}