package services

import (
	"fmt"
	"time"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repository"
)

type Payment struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Amount        int       `json:"amount"`
	TariffID      int       `json:"tariff_id"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

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
	// Проверяем, существует ли пользователь
	_, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Проверяем, существует ли тариф
	_, err = p.TariffRepo.FindByID(tariffID)
	if err != nil {
		return fmt.Errorf("tariff not found: %w", err)
	}

	// Обновляем тариф пользователя
	if err := p.UserRepo.UpdateUserTariff(userID, tariffID); err != nil {
		return fmt.Errorf("failed to update user tariff: %w", err)
	}

	// Обновляем дату окончания подписки
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

func (p *PaymentService) CreatePayment(userID int, amount int, tariffID int, paymentMethod string) error {
	// Проверяем, существует ли пользователь
	_, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Проверяем, существует ли тариф
	_, err = p.TariffRepo.FindByID(tariffID)
	if err != nil {
		return fmt.Errorf("tariff not found: %w", err)
	}

	// Создаем запись о платеже
	payment := &models.Payment{
		UserID:        userID,
		Amount:        amount,
		TariffID:      tariffID,
		PaymentMethod: paymentMethod,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := p.UserRepo.CreatePayment(payment); err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

func (p *PaymentService) GetPaymentsByUserID(userID int) ([]models.Payment, error) {
	return p.UserRepo.GetPaymentsByUserID(userID)
}

func (p *PaymentService) GetPaymentByID(userID int, paymentID string) (*models.Payment, error) {
	return p.UserRepo.GetPaymentByID(userID, paymentID)
}

func (p *PaymentService) UpdatePaymentStatus(userID int, paymentID string, status string) error {
	return p.UserRepo.UpdatePaymentStatus(userID, paymentID, status)
}
