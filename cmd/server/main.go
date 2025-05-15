package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"vpn-backend/config"
	"vpn-backend/internal/handlers"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repository"
	"vpn-backend/internal/services"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database connection
	dbConn, err := gorm.Open(postgres.Open(cfg.DbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Enable detailed logging
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate database schema
	err = dbConn.AutoMigrate(&models.User{}, &models.Tariff{}, &models.Payment{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbConn)
	tariffRepo := repository.NewTariffRepository(dbConn)

	// Initialize services
	authService := services.NewAuthService(userRepo, cfg.JWTSecret)
	if authService == nil {
		log.Fatalf("Failed to initialize AuthService")
	}

	paymentService := services.NewPaymentService(userRepo, tariffRepo)
	if paymentService == nil {
		log.Fatalf("Failed to initialize PaymentService")
	}

	xrayService := services.NewXrayService(userRepo, cfg.XrayConfigPath, cfg.XrayTemplatePath)
	if xrayService == nil {
		log.Fatalf("Failed to initialize XrayService")
	}

	trafficService := services.NewTrafficService(userRepo, paymentService)
	if trafficService == nil {
		log.Fatalf("Failed to initialize TrafficService")
	}

	// Attach Xray service to payment service
	paymentService.AttachXrayService(xrayService)

	// Generate subscription file
	err = handlers.GenerateSubscriptionFile("/root/xray/config.json", "subscription.txt")
	if err != nil {
		log.Fatalf("Failed to generate subscription.txt: %v", err)
	}

	// Initialize handlers
	userHandler := handlers.NewUserHandler(authService, paymentService, xrayService, trafficService) // Pass TrafficService
	adminHandler := handlers.NewAdminHandler(userRepo)
	xrayHandler := handlers.NewXrayHandler(xrayService)
	trafficHandler := handlers.NewTrafficHandler(trafficService) // Initialize TrafficHandler
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	// Initialize router
	r := mux.NewRouter()

	// Middleware
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.RecoveryMiddleware)

	// Public routes
	r.HandleFunc("/register", userHandler.Register).Methods("POST")
	r.HandleFunc("/login", userHandler.Login).Methods("POST")

	// Путь к файлу подписки
	subscriptionFilePath := "subscription.txt" // или относительный путь, если сервер запускается из этой папки

	// HTTP endpoint для отдачи файла подписки
	r.HandleFunc("/subscription.txt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, subscriptionFilePath)
	}).Methods("GET")

	// User routes
	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	userRouter.HandleFunc("/me", userHandler.GetMe).Methods("GET")
	userRouter.HandleFunc("/change-tariff", userHandler.ChangeTariff).Methods("POST")
	userRouter.HandleFunc("/traffic", trafficHandler.GetTraffic).Methods("GET")                        // Add traffic route
	userRouter.HandleFunc("/delete-account", userHandler.DeleteAccount).Methods("POST")                // Add delete account route
	userRouter.HandleFunc("/request-password-reset", userHandler.RequestPasswordReset).Methods("POST") // Add request password reset route
	userRouter.HandleFunc("/payments", paymentHandler.CreatePayment).Methods("POST")
	userRouter.HandleFunc("/payments", paymentHandler.GetUserPayments).Methods("GET")
	userRouter.HandleFunc("/payments/{id}", paymentHandler.GetPaymentByID).Methods("GET")
	userRouter.HandleFunc("/payments/{id}", paymentHandler.UpdatePaymentStatus).Methods("PUT")
	userRouter.HandleFunc("/subscription", userHandler.GetSubscription).Methods("GET")
	userRouter.HandleFunc("/hiddify-config", userHandler.GetHiddifyConfig).Methods("GET")

	// Xray config route
	userRouter.HandleFunc("/config", handlers.NewConfigHandler(xrayService).GetConfig).Methods("GET")

	// Admin routes
	adminRouter := r.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	adminRouter.Use(middleware.AdminOnlyMiddleware(cfg.AdminToken))
	adminRouter.HandleFunc("/users", adminHandler.GetAllUsers).Methods("GET")
	adminRouter.HandleFunc("/ban/{id}", adminHandler.BanUser).Methods("POST")

	// Xray routes
	xrayRouter := r.PathPrefix("/xray").Subrouter()
	xrayRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	xrayRouter.Use(middleware.AdminOnlyMiddleware(cfg.AdminToken))
	xrayRouter.HandleFunc("/reload", xrayHandler.ReloadConfig).Methods("POST")
	xrayRouter.HandleFunc("/restart", xrayHandler.Restart).Methods("POST")

	// CORS setup
	headersOk := gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	originsOk := gorillaHandlers.AllowedOrigins([]string{"*"})
	methodsOk := gorillaHandlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	// HTTP Server configuration
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      gorillaHandlers.CORS(originsOk, headersOk, methodsOk)(r),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	log.Printf("Server listening on port %s", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
