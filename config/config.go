package config

import (
	"log"
	"os"
)

type Config struct {
	DbURL               string
	ServerPort          string
	JWTSecret           string
	AdminToken          string
	TelegramBotUsername string
	TelegramBotToken    string
	XrayConfigPath      string
	XrayTemplatePath    string
	YooKassaShopID      string
	YooKassaSecret      string
	YooKassaReturnURL   string
	FrontendURL         string
	AdminChatIDs        string // comma-separated telegram chat IDs for admin notifications
	PublicOfferURL      string // URL to the public offer/terms
}

func Load() *Config {
	dbURL := getEnv("DATABASE_URL", "postgres://glebasik_k:8915720@localhost:5432/vpn?sslmode=disable")
	serverPort := getEnv("SERVER_PORT", "8081")
	jwtSecret := getEnv("JWT_SECRET", "your-jwt-secret")
	adminToken := getEnv("ADMIN_TOKEN", "admin-token")
	xrayConfigPath := getEnv("XRAY_CONFIG_PATH", "/etc/xray/config.json")
	xrayTemplatePath := getEnv("XRAY_TEMPLATE_PATH", "/etc/xray/config_template.json")
	yooShopID := getEnv("YOOKASSA_SHOP_ID", "")
	yooSecret := getEnv("YOOKASSA_SECRET", "")
	yooReturnURL := getEnv("YOOKASSA_RETURN_URL", "")
	frontendURL := getEnv("FRONTEND_URL", "https://your-frontend.com")
	telegramBot := getEnv("TELEGRAM_BOT_USERNAME", "")
	telegramToken := getEnv("TELEGRAM_BOT_TOKEN", "")
	adminChatIDs := getEnv("ADMIN_CHAT_IDS", "")
	publicOffer := getEnv("PUBLIC_OFFER_URL", "")

	return &Config{
		DbURL:               dbURL,
		ServerPort:          serverPort,
		JWTSecret:           jwtSecret,
		AdminToken:          adminToken,
		TelegramBotUsername: telegramBot,
		TelegramBotToken:    telegramToken,
		XrayConfigPath:      xrayConfigPath,
		XrayTemplatePath:    xrayTemplatePath,
		YooKassaShopID:      yooShopID,
		YooKassaSecret:      yooSecret,
		YooKassaReturnURL:   yooReturnURL,
		FrontendURL:         frontendURL,
		AdminChatIDs:        adminChatIDs,
		PublicOfferURL:      publicOffer,
	}
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Printf("Warning: Environment variable %s not set, using default value: %s", key, defaultValue)
		return defaultValue
	}
	return value
}
