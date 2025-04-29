package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"text/template"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repository"
)

type XrayService struct {
	repo repository.UserRepository
}

func NewXrayService(repo repository.UserRepository) *XrayService {
	return &XrayService{repo: repo}
}

const (
	xrayConfigPath   = "/etc/xray/config.json"
	xrayTemplatePath = "/etc/xray/config_template.json"
	xrayServiceName  = "xray" // можно заменить на имя docker-контейнера
)

func (s *XrayService) AddUser(uuid, email string) error {
	users, err := s.repo.GetAllUsers()
	if err != nil {
		return err
	}
	users = append(users, models.User{UUID: uuid, Email: email})
	return s.RegenerateConfig(users)
}

func (s *XrayService) RemoveUser(uuid string) error {
	users, err := s.repo.GetAllUsers()
	if err != nil {
		return err
	}
	filtered := make([]models.User, 0)
	for _, u := range users {
		if u.UUID != uuid {
			filtered = append(filtered, u)
		}
	}
	return s.RegenerateConfig(filtered)
}

func (s *XrayService) RegenerateConfig(users []models.User) error {
	log.Println("🔄 Генерация Xray конфигурации...")

	tmplContent, err := os.ReadFile(xrayTemplatePath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать шаблон: %w", err)
	}

	tmpl, err := template.New("xray").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("ошибка разбора шаблона: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct{ Users []models.User }{users})
	if err != nil {
		return fmt.Errorf("ошибка генерации файла: %w", err)
	}

	var js map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &js); err != nil {
		return fmt.Errorf("некорректный JSON: %w", err)
	}

	err = os.WriteFile(xrayConfigPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	log.Println("✅ Конфигурация Xray успешно обновлена.")
	return nil
}

func (s *XrayService) RestartXray() error {
	log.Println("🔁 Перезапуск Xray...")
	cmd := exec.Command("systemctl", "restart", xrayServiceName)
	err := cmd.Run()
	if err != nil {
		log.Printf("❌ Ошибка при перезапуске: %v", err)
		return errors.New("не удалось перезапустить Xray")
	}
	log.Println("✅ Xray перезапущен.")
	return nil
}
