package services

import (
	"fmt"
	"os/exec"
	"vpn-backend/internal/repository"
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
	// Формируем JSON-запрос для uplink
	uplinkRequest := fmt.Sprintf(`{"jsonrpc":"2.0","method":"StatsService.QueryStats","params":{"pattern":"user>>>%s>>>traffic>>>uplink","reset":false},"id":1}`, userUUID)

	// Отправляем запрос к Xray API
	cmd := exec.Command("curl", "--http0.9", "--silent", "--output", "-", "-X", "POST", "http://127.0.0.1:10085/stats/query", "-H", "Content-Type: application/json", "-d", uplinkRequest)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute curl command: %w", err)
	}

	// Парсим бинарный ответ
	uplinkTraffic, err := parseTrafficResponse(output)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uplink traffic: %w", err)
	}

	// Аналогично для downlink
	downlinkRequest := fmt.Sprintf(`{"jsonrpc":"2.0","method":"StatsService.QueryStats","params":{"pattern":"user>>>%s>>>traffic>>>downlink","reset":false},"id":1}`, userUUID)
	cmd = exec.Command("curl", "--http0.9", "--silent", "--output", "-", "-X", "POST", "http://127.0.0.1:10085/stats/query", "-H", "Content-Type: application/json", "-d", downlinkRequest)
	output, err = cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute curl command: %w", err)
	}

	downlinkTraffic, err := parseTrafficResponse(output)
	if err != nil {
		return 0, fmt.Errorf("failed to parse downlink traffic: %w", err)
	}

	// Возвращаем суммарный трафик
	return uplinkTraffic + downlinkTraffic, nil
}

// parseTrafficResponse парсит бинарный ответ от Xray API
func parseTrafficResponse(response []byte) (int64, error) {
	// Проверяем длину ответа
	if len(response) < 8 {
		return 0, fmt.Errorf("invalid binary response length")
	}

	// Извлекаем значение трафика (например, 8 байт для int64)
	traffic := int64(response[4])<<24 | int64(response[5])<<16 | int64(response[6])<<8 | int64(response[7])

	return traffic, nil
}

// TrackTrafficUsage records the traffic usage for a user.
func (s *TrafficService) TrackTrafficUsage(userUUID string, bytes int64) error {
	// TODO: Implement the logic to track traffic usage.
	// This might involve updating a traffic tracking service or database.
	_ = userUUID
	_ = bytes
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
