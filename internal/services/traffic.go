package services

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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
	// Get traffic from Xray API or logs
	cmd := exec.Command("xray", "api", "stats", "--server=127.0.0.1:10085", fmt.Sprintf("user>>>%s>>>traffic>>>uplink", userUUID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get traffic stats: %w", err)
	}

	// Parse traffic value
	trafficStr := strings.TrimSpace(string(output))
	traffic, err := strconv.ParseInt(trafficStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse traffic value: %w", err)
	}

	// Get user by UUID
	user, err := s.UserRepo.GetUserByUUID(userUUID)
	if err != nil {
		return 0, err
	}

	// Check traffic limit
	if ok, err := s.PaymentService.CheckTariffLimit(int(user.ID), traffic); err != nil {
		return 0, err
	} else if !ok {
		if err := s.UserRepo.UpdateUsedTraffic(int(user.ID), traffic); err != nil {
			return 0, err
		}
		if err := s.UserRepo.BanUser(int(user.ID), true); err != nil {
			return 0, err
		}
	}

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
