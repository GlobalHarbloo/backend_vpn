package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"text/template"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repository"
)

type XrayService struct {
	Repo         *repository.UserRepository
	ConfigPath   string
	TemplatePath string
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

	// Оставляем только не заблокированных
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

	// Резервная копия текущего конфига
	if err := os.Rename(s.ConfigPath, s.ConfigPath+".bak"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to backup config file: %w", err)
	}

	if err := os.WriteFile(s.ConfigPath, buf.Bytes(), 0644); err != nil {
		_ = os.Rename(s.ConfigPath+".bak", s.ConfigPath)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (s *XrayService) RestartXray() error {
	cmd := exec.Command("systemctl", "restart", "xray")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart Xray: %w, output: %s", err, string(output))
	}
	log.Printf("Xray restarted successfully, output: %s", string(output))
	return nil
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
