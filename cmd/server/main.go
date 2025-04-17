package main

import (
	"log"
	"net/http"
	"vpn-backend/config"
	"vpn-backend/internal/db"
	"vpn-backend/internal/handlers"
	"vpn-backend/internal/repository"
	"vpn-backend/internal/services"

	"github.com/gorilla/mux"
)

func main() {
	// Загружаем конфиг
	cfg := config.Load()

	// Инициализируем базу данных
	err := db.Init(cfg)
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	// Репозиторий
	userRepo := repository.NewUserRepository()

	// Сервисы
	authService := services.NewAuthService(userRepo)
	paymentService := services.NewPaymentService(userRepo)
	xrayService := services.NewXrayService(userRepo)

	// Хендлеры
	userHandler := handlers.NewUserHandler(authService, paymentService)
	adminHandler := handlers.NewAdminHandler(userRepo)
	xrayHandler := handlers.NewXrayHandler(xrayService)

	// Роутер
	r := mux.NewRouter()

	// Пользовательские роуты
	r.HandleFunc("/register", userHandler.Register).Methods("POST")
	r.HandleFunc("/login", userHandler.Login).Methods("POST")

	// Админские роуты
	r.HandleFunc("/admin/users", adminHandler.GetAllUsers).Methods("GET")
	r.HandleFunc("/admin/ban/{id}", adminHandler.BanUser).Methods("POST")

	// Xray
	r.HandleFunc("/xray/restart", xrayHandler.Restart).Methods("POST")
	r.HandleFunc("/xray/reload", xrayHandler.ReloadConfig).Methods("POST")

	log.Println("🚀 Сервер запущен на :8080")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
