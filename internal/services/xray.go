package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"text/template"
	"time"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repository"
)

type XrayService struct {
	Repo         *repository.UserRepository
	ConfigPath   string
	TemplatePath string
	mu           sync.Mutex
}

func NewXrayService(repo *repository.UserRepository, configPath string, templatePath string) *XrayService {
	return &XrayService{
		Repo:         repo,
		ConfigPath:   configPath,
		TemplatePath: templatePath,
	}
}

func (s *XrayService) RegenerateConfig() error {
	users, err := s.Repo.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get all users: %w", err)
	}

	activeUsers := make([]models.User, 0)
	for _, user := range users {
		if !user.IsBanned {
			activeUsers = append(activeUsers, user)
		}
	}

	templateBytes, err := os.ReadFile(s.TemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	tpl, err := template.New("xray").Parse(string(templateBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]interface{}{"Users": activeUsers}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if !json.Valid(buf.Bytes()) {
		return fmt.Errorf("invalid JSON generated")
	}

	if err := os.Rename(s.ConfigPath, s.ConfigPath+".bak"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to backup config file: %w", err)
	}

	if err := os.WriteFile(s.ConfigPath, buf.Bytes(), 0644); err != nil {
		_ = os.Rename(s.ConfigPath+".bak", s.ConfigPath)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (s *XrayService) loadConfig() (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	configBytes, err := os.ReadFile(s.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{
				"inbounds": []interface{}{
					map[string]interface{}{
						"port":     1080,
						"protocol": "vmess",
						"settings": map[string]interface{}{
							"clients": []interface{}{},
						},
					},
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func (s *XrayService) saveConfig(config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.ConfigPath, configBytes, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (s *XrayService) AddUserToConfig(user *models.User) error {
	config, err := s.loadConfig()
	if err != nil {
		log.Printf("Error loading Xray config: %v", err)
		return fmt.Errorf("failed to load Xray config: %w", err)
	}

	// Проверяем, существует ли пользователь
	inbounds := config["inbounds"].([]interface{})
	firstInbound := inbounds[0].(map[string]interface{})
	settings := firstInbound["settings"].(map[string]interface{})
	clients := settings["clients"].([]interface{})

	for _, client := range clients {
		if client.(map[string]interface{})["id"] == user.UUID {
			log.Printf("User with UUID %s already exists in Xray config", user.UUID)
			return fmt.Errorf("user already exists in config")
		}
	}

	// Добавляем нового пользователя
	newClient := map[string]interface{}{
		"id":      user.UUID,
		"email":   user.Email,
		"level":   0,
		"alterId": 0,
	}
	clients = append(clients, newClient)
	settings["clients"] = clients

	if err := s.saveConfig(config); err != nil {
		log.Printf("Error saving Xray config: %v", err)
		return fmt.Errorf("failed to save Xray config: %w", err)
	}

	return nil
}

func (s *XrayService) RemoveUserFromConfig(userUUID string) error {
	config, err := s.loadConfig()
	if err != nil {
		return err
	}

	inbounds := config["inbounds"].([]interface{})
	firstInbound := inbounds[0].(map[string]interface{})
	settings := firstInbound["settings"].(map[string]interface{})
	clients := settings["clients"].([]interface{})

	newClients := []interface{}{}
	for _, client := range clients {
		if client.(map[string]interface{})["id"] != userUUID {
			newClients = append(newClients, client)
		}
	}
	settings["clients"] = newClients

	return s.saveConfig(config)
}

func (s *XrayService) UpdateUserTariff(userUUID string, level int) error {
	config, err := s.loadConfig()
	if err != nil {
		return err
	}

	inbounds := config["inbounds"].([]interface{})
	firstInbound := inbounds[0].(map[string]interface{})
	settings := firstInbound["settings"].(map[string]interface{})
	clients := settings["clients"].([]interface{})

	for _, client := range clients {
		clientMap := client.(map[string]interface{})
		if clientMap["id"] == userUUID {
			clientMap["level"] = level
			break
		}
	}

	return s.saveConfig(config)
}

func (s *XrayService) RestartXray() error {
	cmd := exec.Command("systemctl", "restart", "xray")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to restart Xray: %v, output: %s", err, string(output))
		return fmt.Errorf("failed to restart Xray: %w", err)
	}
	log.Printf("Xray restarted successfully, output: %s", string(output))
	return nil
}

func (s *XrayService) ScheduleRestart() {
	go func() {
		time.Sleep(1 * time.Second)
		if err := s.RestartXray(); err != nil {
			log.Printf("Error restarting Xray: %v", err)
		}
	}()
}

func (s *XrayService) GenerateUserConfig(user *models.User) ([]byte, error) {
	templateBytes, err := os.ReadFile(s.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	tpl, err := template.New("user_xray").Parse(string(templateBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]interface{}{
		"User":   user,
		"Tariff": user.Tariff,
	}); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	if !json.Valid(buf.Bytes()) {
		return nil, fmt.Errorf("invalid JSON generated for user config")
	}

	return buf.Bytes(), nil
}
