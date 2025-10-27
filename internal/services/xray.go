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

	"github.com/yourusername/vpn-backend/internal/models"
	"github.com/yourusername/vpn-backend/internal/repository"
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

// findInboundWithClients returns the first inbound entry that contains a settings.clients array.
// It is more robust than searching by protocol name (vless/vmess) because templates may vary.
func (s *XrayService) findInboundWithClients(config map[string]interface{}) (map[string]interface{}, error) {
	inboundsI, ok := config["inbounds"]
	if !ok || inboundsI == nil {
		return nil, fmt.Errorf("no inbounds in config")
	}
	inbounds, ok := inboundsI.([]interface{})
	if !ok || inbounds == nil || len(inbounds) == 0 {
		return nil, fmt.Errorf("no inbounds found in config")
	}
	for _, ib := range inbounds {
		m, ok := ib.(map[string]interface{})
		if !ok {
			continue
		}
		settingsI, ok := m["settings"]
		if !ok || settingsI == nil {
			continue
		}
		settings, ok := settingsI.(map[string]interface{})
		if !ok || settings == nil {
			continue
		}
		if clientsI, exists := settings["clients"]; exists && clientsI != nil {
			if _, ok := clientsI.([]interface{}); ok {
				return m, nil
			}
		}
	}
	return nil, fmt.Errorf("inbound with clients not found in config")
}

func (s *XrayService) AddUserToConfig(user *models.User) error {
	log.Printf("[Xray] AddUserToConfig called; configPath=%s, user=%s", s.ConfigPath, user.Email)
	config, err := s.loadConfig()
	if err != nil {
		log.Printf("Error loading Xray config: %v", err)
		return fmt.Errorf("failed to load Xray config: %w", err)
	}

	targetInbound, err := s.findInboundWithClients(config)
	if err != nil {
		log.Printf("[Xray] Error finding inbound with clients: %v", err)
		return fmt.Errorf("failed to find inbound with clients: %w", err)
	}
	log.Printf("[Xray] Found inbound for clients; protocol=%v port=%v", targetInbound["protocol"], targetInbound["port"])

	settingsI, ok := targetInbound["settings"]
	if !ok {
		settingsI = map[string]interface{}{"clients": []interface{}{}}
		targetInbound["settings"] = settingsI
	}
	settings, _ := settingsI.(map[string]interface{})

	clientsI, ok := settings["clients"]
	var clients []interface{}
	if ok {
		clients, _ = clientsI.([]interface{})
	} else {
		clients = []interface{}{}
	}
	log.Printf("[Xray] clients before: %d", len(clients))

	for _, client := range clients {
		if cm, ok := client.(map[string]interface{}); ok {
			if cm["id"] == user.UUID {
				log.Printf("[Xray] User with UUID %s already exists in Xray config", user.UUID)
				return fmt.Errorf("user already exists in config")
			}
		}
	}

	newClient := map[string]interface{}{
		"id":      user.UUID,
		"email":   user.Email,
		"level":   0,
		"alterId": 0,
	}
	clients = append(clients, newClient)
	settings["clients"] = clients

	if err := s.saveConfig(config); err != nil {
		log.Printf("[Xray] Error saving Xray config: %v", err)
		return fmt.Errorf("failed to save Xray config: %w", err)
	}

	log.Printf("[Xray] clients after: %d", len(clients))
	log.Printf("[Xray] SaveConfig OK; path=%s", s.ConfigPath)

	return nil
}

func (s *XrayService) RemoveUserFromConfig(userUUID string) error {
	config, err := s.loadConfig()
	if err != nil {
		return err
	}

	targetInbound, err := s.findInboundWithClients(config)
	if err != nil {
		return err
	}

	settingsI, _ := targetInbound["settings"]
	settings, _ := settingsI.(map[string]interface{})
	clientsI, _ := settings["clients"].([]interface{})

	newClients := []interface{}{}
	for _, client := range clientsI {
		if cm, ok := client.(map[string]interface{}); ok {
			if cm["id"] != userUUID {
				newClients = append(newClients, cm)
			}
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

	targetInbound, err := s.findInboundWithClients(config)
	if err != nil {
		return err
	}

	settingsI, _ := targetInbound["settings"]
	settings, _ := settingsI.(map[string]interface{})
	clientsI, _ := settings["clients"].([]interface{})

	for i, client := range clientsI {
		if cm, ok := client.(map[string]interface{}); ok {
			if cm["id"] == userUUID {
				cm["level"] = level
				clientsI[i] = cm
				break
			}
		}
	}
	settings["clients"] = clientsI

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

func (s *XrayService) GetUserConfigFromFile(user *models.User) ([]byte, error) {
	config, err := s.loadConfig()
	if err != nil {
		return nil, err
	}

	targetInbound, err := s.findInboundWithClients(config)
	if err != nil {
		return nil, err
	}

	settingsI, _ := targetInbound["settings"]
	settings, _ := settingsI.(map[string]interface{})
	clientsI, _ := settings["clients"].([]interface{})

	// Найти клиента по UUID
	var userClient map[string]interface{}
	for _, c := range clientsI {
		if client, ok := c.(map[string]interface{}); ok {
			if client["id"] == user.UUID {
				userClient = client
				break
			}
		}
	}
	if userClient == nil {
		return nil, fmt.Errorf("user not found in xray config")
	}

	// Собрать минимальный конфиг для пользователя
	userConfig := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"port":     targetInbound["port"],
				"protocol": targetInbound["protocol"],
				"settings": map[string]interface{}{
					"clients": []interface{}{userClient},
				},
			},
		},
	}

	return json.MarshalIndent(userConfig, "", "  ")
}

func (s *XrayService) CheckUserInConfig(userUUID string) (bool, error) {
	config, err := s.loadConfig()
	if err != nil {
		return false, err
	}
	targetInbound, err := s.findInboundWithClients(config)
	if err != nil {
		return false, err
	}
	settingsI, _ := targetInbound["settings"]
	settings, _ := settingsI.(map[string]interface{})
	clientsI, _ := settings["clients"].([]interface{})
	for _, client := range clientsI {
		if cm, ok := client.(map[string]interface{}); ok {
			if cm["id"] == userUUID {
				return true, nil
			}
		}
	}
	return false, nil
}
