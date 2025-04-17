package db

import (
	"database/sql"
	"fmt"
	"vpn-backend/config"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(cfg *config.Config) error {
	var err error
	DB, err = sql.Open("postgres", cfg.DbURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %w", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	fmt.Println("✅ Успешно подключено к базе данных")
	return nil
}
