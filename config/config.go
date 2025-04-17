package config

import (
	"log"
	"os"
)

type Config struct {
	DbURL      string
	ServerPort string
}

func Load() *Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/vpn?sslmode=disable"
		log.Println("⚠️  Используется значение БД по умолчанию")
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
		log.Println("⚠️  Используется порт сервера по умолчанию")
	}

	return &Config{
		DbURL:      dbURL,
		ServerPort: port,
	}
}
