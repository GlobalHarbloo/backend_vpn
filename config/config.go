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
	RobokassaLogin      string
	RobokassaPassword1  string
	RobokassaPassword2  string
	DefaultLang         string
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
	// NOTE: Defaults below are provided per request. In production it's safer
	// to set these via environment variables or a secrets manager rather than
	// committing secrets to source code.
	robokassaLogin := getEnv("ROBOKASSA_LOGIN", "freshvpn")
	// Password #1 — used to sign redirect requests (MD5 signature)
	robokassaP1 := getEnv("ROBOKASSA_PASSWORD1", "gdId0uf28sfAXny9G0DN")
	// Password #2 — used to validate server-to-server notifications (Result URL)
	robokassaP2 := getEnv("ROBOKASSA_PASSWORD2", "CP1H0EAI6PadIWl4iQ4H")
	// Frontend / merchant return URLs. Provided value here uses the Telegram link
	// supplied; normally this should be your frontend URL (e.g. https://example.com)
	frontendURL := getEnv("FRONTEND_URL", "http://t.me/Fresh_v_p_n")
	defaultLang := getEnv("DEFAULT_LANG", "ru")
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
		RobokassaLogin:      robokassaLogin,
		RobokassaPassword1:  robokassaP1,
		RobokassaPassword2:  robokassaP2,
		DefaultLang:         defaultLang,
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
