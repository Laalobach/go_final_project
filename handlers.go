package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

// Глобальные константы
const (
    dateFormat = "20060102"          // Формат даты
    defaultLimit = 50                // Лимит записей по умолчанию
)

// Task представляет структуру задачи
type Task struct {
    ID      string `json:"id,omitempty"` // omitempty: поле будет пропущено, если пустое
    Date    string `json:"date"`
    Title   string `json:"title"`
    Comment string `json:"comment,omitempty"` // omitempty: поле будет пропущено, если пустое
    Repeat  string `json:"repeat,omitempty"`  // omitempty: поле будет пропущено, если пустое
    Error   string `json:"error,omitempty"`   // Для возврата ошибок
}

// TasksResponse представляет структуру ответа с задачами
type TasksResponse struct {
    Tasks []Task `json:"tasks"`
    Error string `json:"error,omitempty"`
}

func nextDateHandler(w http.ResponseWriter, r *http.Request) {
    // Получаем параметры из запроса
    nowStr := r.FormValue("now")
    dateStr := r.FormValue("date")
    repeat := r.FormValue("repeat")

    // Парсим текущее время
    now, err := time.Parse(dateFormat, nowStr)
    if err != nil {
        http.Error(w, "неверный формат now: "+nowStr, http.StatusBadRequest)
        return
    }

    // Вычисляем следующую дату
    nextDate, err := NextDate(now, dateStr, repeat)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Возвращаем результат
    if nextDate == "" {
        w.WriteHeader(http.StatusNoContent)
    } else {
        w.Write([]byte(nextDate))
    }
}

func taskHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")

    if db == nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(Task{Error: "База данных не инициализирована"})
        return
    }

    switch r.Method {
    case http.MethodGet:
        // Обработка GET-запроса для получения задачи по ID
        id := r.URL.Query().Get("id")
        if id == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(Task{Error: "Не указан идентификатор"})
            return
        }

        var task Task
        var date string
        err := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).Scan(&task.ID, &date, &task.Title, &task.Comment, &task.Repeat)
        if err != nil {
            if err == sql.ErrNoRows {
                w.WriteHeader(http.StatusNotFound)
                json.NewEncoder(w).Encode(Task{Error: "Задача не найдена"})
            } else {
                w.WriteHeader(http.StatusInternalServerError)
                json.NewEncoder(w).Encode(Task{Error: "Ошибка при получении задачи"})
            }
            return
        }

        task.Date = date
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(task)

    case http.MethodPost:
        // Обработка POST-запроса для добавления новой задачи
        var task Task
        if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка десериализации JSON"})
            return
        }

        if task.Title == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(Task{Error: "Не указан заголовок задачи"})
            return
        }

        var date time.Time
        var err error
        if task.Date == "" {
            date = time.Now()
        } else {
            date, err = time.Parse(dateFormat, task.Date)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(Task{Error: "Неверный формат даты"})
                return
            }
        }

        // Исправленное сравнение дат (без времени)
        now := time.Now()
        nowFormatted := now.Format(dateFormat)
        dateFormatted := date.Format(dateFormat)

        if dateFormatted < nowFormatted {
            if task.Repeat == "" {
                // Устанавливаем сегодняшнюю дату (без времени)
                date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
            } else {
                nextDateStr, err := NextDate(now, dateFormatted, task.Repeat)
                if err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(Task{Error: err.Error()})
                    return
                }
                date, _ = time.Parse(dateFormat, nextDateStr)
            }
        }

        // Проверка повторения (если нужно)
        if task.Repeat != "" {
            _, err := NextDate(now, date.Format(dateFormat), task.Repeat)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(Task{Error: "Неверный формат правила повторения"})
                return
            }
        }

        res, err := db.Exec(
            "INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)",
            date.Format(dateFormat), task.Title, task.Comment, task.Repeat,
        )
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при добавлении задачи"})
            return
        }

        id, err := res.LastInsertId()
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при получении идентификатора задачи"})
            return
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(Task{ID: fmt.Sprintf("%d", id)})

    case http.MethodPut:
        // Обработка PUT-запроса для обновления задачи
        var task Task
        if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка десериализации JSON"})
            return
        }

        if task.ID == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(Task{Error: "Не указан идентификатор задачи"})
            return
        }

        if task.Title == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(Task{Error: "Не указан заголовок задачи"})
            return
        }

        var date time.Time
        var err error
        if task.Date == "" {
            date = time.Now()
        } else {
            date, err = time.Parse(dateFormat, task.Date)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(Task{Error: "Неверный формат даты"})
                return
            }
        }

        // Исправленное сравнение для PUT-запроса
        now := time.Now()
        nowFormatted := now.Format(dateFormat)
        dateFormatted := date.Format(dateFormat)

        if dateFormatted < nowFormatted {
            if task.Repeat == "" {
                date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
            } else {
                nextDateStr, err := NextDate(now, dateFormatted, task.Repeat)
                if err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(Task{Error: err.Error()})
                    return
                }
                date, _ = time.Parse(dateFormat, nextDateStr)
            }
        }

        if task.Repeat != "" {
            _, err := NextDate(now, date.Format(dateFormat), task.Repeat)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(Task{Error: "Неверный формат правила повторения"})
                return
            }
        }

        res, err := db.Exec(
            "UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?",
            date.Format(dateFormat), task.Title, task.Comment, task.Repeat, task.ID,
        )
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при обновлении задачи"})
            return
        }

        rowsAffected, err := res.RowsAffected()
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при проверке обновления задачи"})
            return
        }

        if rowsAffected == 0 {
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(Task{Error: "Задача не найдена"})
            return
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(struct{}{})

    case http.MethodDelete:
        // Обработка DELETE-запроса для удаления задачи
        id := r.URL.Query().Get("id")
        if id == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(Task{Error: "Не указан идентификатор задачи"})
            return
        }

        // Удаляем задачу из базы данных
        res, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при удалении задачи"})
            return
        }

        // Проверяем, была ли удалена задача
        rowsAffected, err := res.RowsAffected()
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при проверке удаления задачи"})
            return
        }

        if rowsAffected == 0 {
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(Task{Error: "Задача не найдена"})
            return
        }

        // Возвращаем пустой JSON в случае успешного удаления
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(struct{}{})

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(Task{Error: "Метод не поддерживается"})
    }
}

func tasksHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")

    if db == nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(TasksResponse{Error: "База данных не инициализирована"})
        return
    }

    search := r.URL.Query().Get("search")
    var query string
    var args []interface{}

    if search == "" {
        query = fmt.Sprintf("SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT %d", defaultLimit)
    } else {
        date, err := time.Parse("02.01.2006", search)
        if err == nil {
            query = fmt.Sprintf("SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? ORDER BY date LIMIT %d", defaultLimit)
            args = append(args, date.Format(dateFormat))
        } else {
            query = fmt.Sprintf("SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT %d", defaultLimit)
            searchPattern := "%" + search + "%"
            args = append(args, searchPattern, searchPattern)
        }
    }

    rows, err := db.Query(query, args...)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(TasksResponse{Error: "Ошибка при выполнении запроса"})
        return
    }
    defer rows.Close()

    var tasks []Task
    for rows.Next() {
        var task Task
        var id int64
        var date string

        if err := rows.Scan(&id, &date, &task.Title, &task.Comment, &task.Repeat); err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(TasksResponse{Error: "Ошибка при чтении данных"})
            return
        }

        task.ID = fmt.Sprintf("%d", id)
        task.Date = date
        tasks = append(tasks, task)
    }

    // Проверяем ошибки после завершения итерации
    if err := rows.Err(); err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(TasksResponse{Error: "Ошибка при обработке результатов запроса"})
        return
    }

    if tasks == nil {
        tasks = []Task{}
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(TasksResponse{Tasks: tasks})
}

func taskDoneHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")

    if db == nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(Task{Error: "База данных не инициализирована"})
        return
    }

    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(Task{Error: "Метод не поддерживается"})
        return
    }

    id := r.URL.Query().Get("id")
    if id == "" {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(Task{Error: "Не указан идентификатор задачи"})
        return
    }

    var task Task
    var date string
    err := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?", id).Scan(&task.ID, &date, &task.Title, &task.Comment, &task.Repeat)
    if err != nil {
        if err == sql.ErrNoRows {
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(Task{Error: "Задача не найдена"})
        } else {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при получении задачи"})
        }
        return
    }

    task.Date = date

    if task.Repeat == "" {
        // Если задача одноразовая, удаляем её
        _, err := db.Exec("DELETE FROM scheduler WHERE id = ?", id)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при удалении задачи"})
            return
        }
    } else {
        // Если задача периодическая, обновляем дату
        now := time.Now()
        nextDate, err := NextDate(now, task.Date, task.Repeat)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при расчете следующей даты"})
            return
        }

        _, err = db.Exec("UPDATE scheduler SET date = ? WHERE id = ?", nextDate, id)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(Task{Error: "Ошибка при обновлении задачи"})
            return
        }
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(struct{}{})
}