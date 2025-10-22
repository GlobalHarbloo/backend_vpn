package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"
	"github.com/yourusername/vpn-backend/internal/repository"
)

type TrafficService struct {
	UserRepo       *repository.UserRepository
	PaymentService *PaymentService
}

func NewTrafficService(userRepo *repository.UserRepository, paymentService *PaymentService) *TrafficService {
	return &TrafficService{
		UserRepo:       userRepo,
		PaymentService: paymentService,
	}
}

// GetUserTraffic returns the total traffic used by a user.
func (s *TrafficService) GetUserTraffic(userUUID string) (int64, error) {
	user, err := s.UserRepo.GetUserByUUID(userUUID)
	if err != nil {
		return 0, fmt.Errorf("user not found: %w", err)
	}
	// Сброс трафика раз в месяц
	monthAgo := time.Now().AddDate(0, -1, 0)
	if user.CreatedAt.Before(monthAgo) && user.UsedTraffic > 0 {
		err := s.UserRepo.UpdateUsedTraffic(int(user.ID), 0)
		if err != nil {
			log.Printf("[Traffic] Error resetting traffic for user %s: %v", userUUID, err)
		}
		return 0, nil
	}

	// Если Xray API недоступен, возвращаем сохраненный трафик из БД
	if user.UsedTraffic > 0 {
		return user.UsedTraffic, nil
	}

	getSum := func(identifier string) int64 {
		uplinkRequest := fmt.Sprintf(`{"jsonrpc":"2.0","method":"StatsService.QueryStats","params":{"pattern":"user>>>%s>>>traffic>>>uplink","reset":false},"id":1}`, identifier)
		cmd := exec.Command("curl", "--http0.9", "--silent", "--output", "-", "-X", "POST", "http://127.0.0.1:10085/stats/query", "-H", "Content-Type: application/json", "-d", uplinkRequest)
		output, err := cmd.Output()
		if err != nil {
			return 0
		}
		uplinkTraffic, err := parseTrafficResponse(output)
		if err != nil {
			uplinkTraffic = 0
		}

		downlinkRequest := fmt.Sprintf(`{"jsonrpc":"2.0","method":"StatsService.QueryStats","params":{"pattern":"user>>>%s>>>traffic>>>downlink","reset":false},"id":1}`, identifier)
		cmd = exec.Command("curl", "--http0.9", "--silent", "--output", "-", "-X", "POST", "http://127.0.0.1:10085/stats/query", "-H", "Content-Type: application/json", "-d", downlinkRequest)
		output, err = cmd.Output()
		if err != nil {
			return uplinkTraffic
		}
		downlinkTraffic, err := parseTrafficResponse(output)
		if err != nil {
			downlinkTraffic = 0
		}
		return uplinkTraffic + downlinkTraffic
	}

	byEmail := getSum(user.Email)
	byUUID := getSum(user.UUID)
	if byEmail >= byUUID {
		return byEmail, nil
	}
	return byUUID, nil
}

// parseTrafficResponse парсит бинарный ответ от Xray API
func parseTrafficResponse(response []byte) (int64, error) {
	// Парсим JSON-ответ от Xray
	type xrayStat struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Stat struct {
				Name  string `json:"name"`
				Value int64  `json:"value"`
			} `json:"stat"`
		} `json:"result"`
	}
	var stat xrayStat
	err := json.Unmarshal(response, &stat)
	if err != nil {
		return 0, fmt.Errorf("failed to parse Xray JSON response: %w", err)
	}
	return stat.Result.Stat.Value, nil
}

// TrackTrafficUsage records the traffic usage for a user.
func (s *TrafficService) TrackTrafficUsage(userUUID string, bytes int64) error {
	user, err := s.UserRepo.GetUserByUUID(userUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Обновляем использованный трафик в базе данных
	newTraffic := user.UsedTraffic + bytes
	if err := s.UserRepo.UpdateUsedTraffic(int(user.ID), newTraffic); err != nil {
		return fmt.Errorf("failed to update traffic: %w", err)
	}

	log.Printf("[Traffic] Updated traffic for user %s: +%d bytes, total: %d", userUUID, bytes, newTraffic)
	return nil
}

// CheckTrafficLimits checks if the user has exceeded the traffic limits of their current tariff.
func (s *TrafficService) CheckTrafficLimits(userID int) (bool, error) {
	// Delegate the traffic limit check to the PaymentService
	exceeded, err := s.PaymentService.CheckTariffLimits(userID)
	if err != nil {
		return false, fmt.Errorf("failed to check traffic limits: %w", err)
	}
	return exceeded, nil
}
