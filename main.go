package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
)

var db *sql.DB // Определяем переменную db

func main() {
	// Инициализация базы данных
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Ошибка при инициализации базы данных: %v", err)
	}
	defer db.Close()

	// Определяем порт, который будет слушать сервер
	port := getPort()

	// Указываем директорию, из которой будем раздавать файлы
	webDir := "./web"

	// Настраиваем обработчик для статических файлов
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	// Регистрируем API-обработчики
	http.HandleFunc("/api/nextdate", nextDateHandler)
	http.HandleFunc("/api/task", taskHandler)
	http.HandleFunc("/api/tasks", tasksHandler)
	http.HandleFunc("/api/task/done", taskDoneHandler)

	// Запускаем сервер
	log.Printf("Сервер запущен на http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Функция для получения порта из переменной окружения или использования значения по умолчанию
func getPort() string {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540" // Порт по умолчанию
	}
	return port
}