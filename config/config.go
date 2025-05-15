package config

import (
	"log"
	"os"
)

type Config struct {
	DbURL            string
	ServerPort       string
	JWTSecret        string
	AdminToken       string
	XrayConfigPath   string
	XrayTemplatePath string
}

func Load() *Config {
	dbURL := getEnv("DATABASE_URL", "postgres://glebasik_k:8915720@localhost:5432/vpn?sslmode=disable")
	serverPort := getEnv("SERVER_PORT", "8081")
	jwtSecret := getEnv("JWT_SECRET", "your-jwt-secret")
	adminToken := getEnv("ADMIN_TOKEN", "admin-token")
	xrayConfigPath := getEnv("XRAY_CONFIG_PATH", "/etc/xray/config.json")
	xrayTemplatePath := getEnv("XRAY_TEMPLATE_PATH", "/etc/xray/config_template.json")

	return &Config{
		DbURL:            dbURL,
		ServerPort:       serverPort,
		JWTSecret:        jwtSecret,
		AdminToken:       adminToken,
		XrayConfigPath:   xrayConfigPath,
		XrayTemplatePath: xrayTemplatePath,
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
