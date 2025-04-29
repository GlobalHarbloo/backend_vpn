package services

import (
	"encoding/json"
	"testing"
	"vpn-backend/internal/models"
)

func TestGenerateUserConfig(t *testing.T) {
	service := &XrayService{
		ConfigPath:   "test_config.json",
		TemplatePath: "test_template.json",
	}

	user := &models.User{
		Email: "test@example.com",
		UUID:  "test-uuid",
	}

	config, err := service.GenerateUserConfig(user)
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	// Проверяем валидность JSON
	var js map[string]interface{}
	if err := json.Unmarshal(config, &js); err != nil {
		t.Fatalf("Invalid JSON generated: %v", err)
	}

	// Проверяем наличие необходимых полей
	inbounds, ok := js["inbounds"].([]interface{})
	if !ok || len(inbounds) == 0 {
		t.Fatal("No inbounds in config")
	}

	firstInbound := inbounds[0].(map[string]interface{})
	settings := firstInbound["settings"].(map[string]interface{})
	clients := settings["clients"].([]interface{})
	if len(clients) == 0 {
		t.Fatal("No clients in config")
	}

	client := clients[0].(map[string]interface{})
	if client["id"] != user.UUID {
		t.Fatalf("Expected UUID %s, got %s", user.UUID, client["id"])
	}
}
