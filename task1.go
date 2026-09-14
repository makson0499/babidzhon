package main

import (
	"fmt"
	"log"
	"net/http"
)

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "Добро пожаловать на наш HTTP-сервер!")
}

func about(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Учебный HTTP-сервер на языке Go с использованием пакета net/http.")
	fmt.Fprintln(w, "⠀⣀⣀⣤⣤⣤⣤⣄⣀⣀⠀⠀⢀⣀⣀⣠⣤⣤⣤⣴⣤⣀")
	fmt.Fprintln(w, "⠘⠟⠋⢉⣠⣤⣭⣭⠭⡍⠀⠀⠀⢩⠭⣭⣭⣤⣄⡈⠙⠻⠂")
	fmt.Fprintln(w, "⠀⠀⠴⠯⢽⣿⡿⣹⣍⠟⠀⠀⠀⠻⠸⠏⠿⠿⠧⡽⠂⠀⠀")
	fmt.Fprintln(w, "⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀")
	fmt.Fprintln(w, "⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠆⠀")
	fmt.Fprintln(w, "⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠀⢰⠃⠀⠀")
	fmt.Fprintln(w, "⠀⠀⠀⠀⠀⢠⣀⣀⣀⣀⣀⣀⣀⣀⡤⠤⠞⠁⢀⡏⠀⠀⠀")
	fmt.Fprintln(w, "⠀⠀⠀⠀⠀⠈⠋⠉⠉⠉⠉⠉⠁⠀⠀⠀⠀⠀⠸⠀⠀⠀⠀")
}

func ping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "pong")
}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/about", about)
	http.HandleFunc("/ping", ping)

	port := ":8080"
	fmt.Printf("Сервер запущен на http://localhost%s\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
