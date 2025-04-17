package services

import (
	"encoding/json"
	"os"
	"os/exec"
	"vpn-backend/internal/repository"
)

const xrayConfigPath = "/etc/xray/config.json"

type XrayService struct {
	Repo *repository.UserRepository
}

type InboundUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type XrayConfig struct {
	Inbound struct {
		Settings struct {
			Clients []InboundUser `json:"clients"`
		} `json:"settings"`
	} `json:"inbounds"`
}

func NewXrayService(repo *repository.UserRepository) *XrayService {
	return &XrayService{Repo: repo}
}

func (s *XrayService) RegenerateConfig() error {
	users, err := s.Repo.GetAllUsers()
	if err != nil {
		return err
	}

	var clients []InboundUser
	for _, u := range users {
		if u.IsBanned {
			continue
		}
		clients = append(clients, InboundUser{ID: u.UUID, Email: u.Email})
	}

	config := XrayConfig{}
	config.Inbound.Settings.Clients = clients

	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(xrayConfigPath, bytes, 0644)
}

func (s *XrayService) RestartXray() error {
	cmd := exec.Command("systemctl", "restart", "xray")
	return cmd.Run()
}
