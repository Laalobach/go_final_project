package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

const (
	dbFileName    = "scheduler.db"
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS scheduler (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		title TEXT NOT NULL,
		comment TEXT,
		repeat TEXT CHECK(LENGTH(repeat) <= 128)
	);
	CREATE INDEX IF NOT EXISTS idx_date ON scheduler (date);
	`
)

// initDB инициализирует базу данных и создает таблицу, если её нет
func initDB() (*sql.DB, error) {
	// Используем фиксированный путь к файлу базы данных
	dbFile := "./scheduler.db"
	log.Printf("Путь к файлу базы данных: %s", dbFile)

	// Открываем базу данных
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть базу данных: %v", err)
	}

	// Проверяем соединение с базой данных
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %v", err)
	}

	// Создаем таблицу и индекс, если они не существуют
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать таблицу: %v", err)
	}

	log.Println("База данных успешно подключена и таблица создана (если не существовала).")
	return db, nil
}