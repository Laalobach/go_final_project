package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, dateStr string, repeat string) (string, error) {
	log.Printf("\n[NextDate] Start calculation")
	log.Printf("[Input] now: %s, dateStr: %s, repeat: %s", now.Format(dateFormat), dateStr, repeat)

	if dateStr == "" {
		log.Print("[Result] Empty date string")
		return "", nil
	}

	date, err := time.Parse(dateFormat, dateStr)
	if err != nil {
		log.Printf("[Error] Date parse failed: %v", err)
		return "", fmt.Errorf("неверный формат даты: %s", dateStr)
	}

	nowFormatted := now.Format(dateFormat)
	dateFormatted := date.Format(dateFormat)

	if repeat == "" {
		if dateFormatted > nowFormatted {
			return dateStr, nil
		}
		return "", nil
	}

	parts := strings.Fields(repeat)
	if len(parts) == 0 {
		return "", fmt.Errorf("неверный формат правила: %s", repeat)
	}

	switch parts[0] {
	case "d":
		if len(parts) != 2 {
			return "", fmt.Errorf("неверный формат правила: %s", repeat)
		}

		days, err := strconv.Atoi(parts[1])
		if err != nil || days < 1 || days > 400 {
			return "", fmt.Errorf("недопустимое количество дней: %s", parts[1])
		}

		nextDate := date
		// Применяем правило хотя бы один раз
		nextDate = nextDate.AddDate(0, 0, days)
		// Продолжаем применять, пока дата не станет строго больше текущей
		for nextDate.Format(dateFormat) <= nowFormatted {
			nextDate = nextDate.AddDate(0, 0, days)
		}

		return nextDate.Format(dateFormat), nil

	case "y":
		nextDate := date
		// Применяем правило хотя бы один раз
		nextDate = nextDate.AddDate(1, 0, 0)
		// Обработка 29 февраля
		if date.Day() == 29 && date.Month() == 2 {
			if !isLeap(nextDate.Year()) {
				nextDate = time.Date(nextDate.Year(), 3, 1, 0, 0, 0, 0, nextDate.Location())
			}
		}
		// Продолжаем применять, пока дата не станет строго больше текущей
		for nextDate.Format(dateFormat) <= nowFormatted {
			nextDate = nextDate.AddDate(1, 0, 0)
			// Корректировка для 29 февраля
			if date.Day() == 29 && date.Month() == 2 && !isLeap(nextDate.Year()) {
				nextDate = time.Date(nextDate.Year(), 3, 1, 0, 0, 0, 0, nextDate.Location())
			}
		}

		return nextDate.Format(dateFormat), nil

	default:
		return "", fmt.Errorf("неподдерживаемый формат правила: %s", parts[0])
	}
}

// Проверка на високосный год
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}