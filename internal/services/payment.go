package services

import (
	"fmt"
	"time"
	"vpn-backend/internal/repository"
)

type PaymentService struct {
	UserRepo   *repository.UserRepository
	TariffRepo *repository.TariffRepository
	Xray       *XrayService
}

func NewPaymentService(userRepo *repository.UserRepository, tariffRepo *repository.TariffRepository) *PaymentService {
	return &PaymentService{
		UserRepo:   userRepo,
		TariffRepo: tariffRepo,
	}
}

func (p *PaymentService) GetTariffExpiry(userID int) (time.Time, error) {
	return p.UserRepo.GetTariffExpiry(userID)
}

func (p *PaymentService) AttachXrayService(x *XrayService) {
	p.Xray = x
}

func (p *PaymentService) XrayService() *XrayService {
	return p.Xray
}

func (p *PaymentService) ChangeTariff(userID int, tariffID int) error {
	_, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	_, err = p.TariffRepo.FindByID(tariffID)
	if err != nil {
		return fmt.Errorf("tariff not found: %w", err)
	}

	if err := p.UserRepo.UpdateUserTariff(userID, tariffID); err != nil {
		return fmt.Errorf("failed to update user tariff: %w", err)
	}

	// Обновляем дату окончания подписки после смены тарифа
	expiry := time.Now().AddDate(0, 1, 0)
	if err := p.UserRepo.UpdateTariffExpiry(userID, expiry); err != nil {
		return fmt.Errorf("failed to update tariff expiry: %w", err)
	}

	return nil
}

func (p *PaymentService) AutoRenewSubscription(userID int) error {
	user, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	newExpiry := user.TariffExpiresAt.AddDate(0, 1, 0)
	if err := p.UserRepo.UpdateTariffExpiry(userID, newExpiry); err != nil {
		return fmt.Errorf("failed to update tariff expiry: %w", err)
	}

	return nil
}

func (p *PaymentService) CheckTariffLimit(userID int, traffic int64) (bool, error) {
	user, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	tariff, err := p.TariffRepo.FindByID(user.TariffID)
	if err != nil {
		return false, fmt.Errorf("tariff not found: %w", err)
	}

	return traffic <= tariff.TrafficLimit, nil
}

func (p *PaymentService) CheckTariffLimits(userID int) (bool, error) {
	user, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	tariff, err := p.TariffRepo.FindByID(user.TariffID)
	if err != nil {
		return false, fmt.Errorf("tariff not found: %w", err)
	}

	// Предположим, что сравниваем used_traffic и traffic_limit
	return user.UsedTraffic <= tariff.TrafficLimit, nil
}
