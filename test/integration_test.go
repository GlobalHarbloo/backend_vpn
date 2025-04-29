package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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

var (
	dbConn         *gorm.DB
	cfg            *config.Config
	userRepo       *repository.UserRepository
	tariffRepo     *repository.TariffRepository
	authService    *services.AuthService
	paymentService *services.PaymentService
	xrayService    *services.XrayService
	trafficService *services.TrafficService
	userHandler    *handlers.UserHandler
	adminHandler   *handlers.AdminHandler
	xrayHandler    *handlers.XrayHandler
	trafficHandler *handlers.TrafficHandler
	router         *mux.Router
)

func setup() {
	// Load configuration
	cfg = config.Load()

	// Initialize database connection
	var err error
	dbConn, err = gorm.Open(postgres.Open(cfg.DbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // Enable detailed logging
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate database schema
	err = dbConn.AutoMigrate(&models.User{}, &models.Tariff{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	// Initialize repositories
	userRepo = repository.NewUserRepository(dbConn)
	tariffRepo = repository.NewTariffRepository(dbConn)

	// Initialize services
	authService = services.NewAuthService(userRepo, cfg.JWTSecret)
	paymentService = services.NewPaymentService(userRepo, tariffRepo)
	xrayService = services.NewXrayService(userRepo, cfg.XrayConfigPath, cfg.XrayTemplatePath)
	trafficService = services.NewTrafficService(userRepo, paymentService)

	// Attach Xray service to payment service
	paymentService.AttachXrayService(xrayService)

	// Initialize handlers
	userHandler = handlers.NewUserHandler(authService, paymentService, xrayService, trafficService)
	adminHandler = handlers.NewAdminHandler(userRepo)
	xrayHandler = handlers.NewXrayHandler(xrayService)
	trafficHandler = handlers.NewTrafficHandler(trafficService)

	// Initialize router
	router = mux.NewRouter()

	// Middleware
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.RecoveryMiddleware)

	// Public routes
	router.HandleFunc("/register", userHandler.Register).Methods("POST")
	router.HandleFunc("/login", userHandler.Login).Methods("POST")

	// User routes
	userRouter := router.PathPrefix("/user").Subrouter()
	userRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	userRouter.HandleFunc("/me", userHandler.GetMe).Methods("GET")
	userRouter.HandleFunc("/change-tariff", userHandler.ChangeTariff).Methods("POST")
	userRouter.HandleFunc("/traffic", trafficHandler.GetTraffic).Methods("GET")
	userRouter.HandleFunc("/delete-account", userHandler.DeleteAccount).Methods("POST")
	userRouter.HandleFunc("/request-password-reset", userHandler.RequestPasswordReset).Methods("POST")

	// Xray config route
	userRouter.HandleFunc("/config", handlers.NewConfigHandler(xrayService).GetConfig).Methods("GET")

	// Admin routes
	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	adminRouter.Use(middleware.AdminOnlyMiddleware(cfg.AdminToken))
	adminRouter.HandleFunc("/users", adminHandler.GetAllUsers).Methods("GET")
	adminRouter.HandleFunc("/ban/{id}", adminHandler.BanUser).Methods("POST")

	// Xray routes
	xrayRouter := router.PathPrefix("/xray").Subrouter()
	xrayRouter.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	xrayRouter.Use(middleware.AdminOnlyMiddleware(cfg.AdminToken))
	xrayRouter.HandleFunc("/reload", xrayHandler.ReloadConfig).Methods("POST")
	xrayRouter.HandleFunc("/restart", xrayHandler.Restart).Methods("POST")
}

func TestMain(m *testing.M) {
	setup()
	exitCode := m.Run()
	// teardown() // Clean up after tests if needed
	os.Exit(exitCode)
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		gorillaHandlers.AllowedOrigins([]string{"*"}),
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "OPTIONS"}),
	)(router)

	handler.ServeHTTP(rr, req)
	return rr
}

func TestRegisterAndLogin(t *testing.T) {
	// Register
	registerData := map[string]string{"email": "test@example.com", "password": "password"}
	registerBody, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := executeRequest(registerReq)

	if registerResp.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusCreated, registerResp.Code, registerResp.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(registerResp.Body.Bytes(), &user); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	// Login
	loginData := map[string]string{"email": "test@example.com", "password": "password"}
	loginBody, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := executeRequest(loginReq)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, loginResp.Code, loginResp.Body.String())
	}

	var tokenData map[string]string
	if err := json.Unmarshal(loginResp.Body.Bytes(), &tokenData); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if _, ok := tokenData["token"]; !ok {
		t.Fatal("Token not found in response")
	}

	token := tokenData["token"]

	// Get Me
	meReq, _ := http.NewRequest("GET", "/user/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meResp := executeRequest(meReq)

	if meResp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, meResp.Code, meResp.Body.String())
	}

	var meData map[string]interface{}
	if err := json.Unmarshal(meResp.Body.Bytes(), &meData); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if meData["email"] != "test@example.com" {
		t.Fatalf("Expected email %s, got %s", "test@example.com", meData["email"])
	}

	// Delete Account
	deleteReq, _ := http.NewRequest("POST", "/user/delete-account", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteResp := executeRequest(deleteReq)

	if deleteResp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, deleteResp.Code, deleteResp.Body.String())
	}

	var deleteData map[string]string
	if err := json.Unmarshal(deleteResp.Body.Bytes(), &deleteData); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if deleteData["status"] != "account deleted" {
		t.Fatalf("Expected status %s, got %s", "account deleted", deleteData["status"])
	}
}

func teardown() {
	// Clean up the database after tests
	if dbConn != nil {
		db, err := dbConn.DB()
		if err != nil {
			log.Fatalf("Failed to get raw DB connection: %v", err)
		}
		db.Close()
	}
	os.Remove("test.db") // Remove the test database file
}

func TestTrafficEndpoint(t *testing.T) {
	// Register a user
	registerData := map[string]string{"email": "traffic@example.com", "password": "password"}
	registerBody, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := executeRequest(registerReq)

	if registerResp.Code != http.StatusCreated {
		t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusCreated, registerResp.Code, registerResp.Body.String())
	}

	// Login the user
	loginData := map[string]string{"email": "traffic@example.com", "password": "password"}
	loginBody, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := executeRequest(loginReq)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, loginResp.Code, loginResp.Body.String())
	}

	var tokenData map[string]string
	if err := json.Unmarshal(loginResp.Body.Bytes(), &tokenData); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	token := tokenData["token"]

	// Call the traffic endpoint
	trafficReq, _ := http.NewRequest("GET", "/user/traffic", nil)
	trafficReq.Header.Set("Authorization", "Bearer "+token)
	trafficResp := executeRequest(trafficReq)

	if trafficResp.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d. Body: %s", http.StatusOK, trafficResp.Code, trafficResp.Body.String())
	}

	// Parse the response
	var trafficData map[string]interface{}
	if err := json.Unmarshal(trafficResp.Body.Bytes(), &trafficData); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	// Assert that the traffic data is present
	if _, ok := trafficData["traffic"]; !ok {
		t.Fatalf("Expected traffic data to be present")
	}

	fmt.Printf("Traffic data: %v\n", trafficData)
}
