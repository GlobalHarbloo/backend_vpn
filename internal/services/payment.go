package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/vpn-backend/internal/models"
	"github.com/yourusername/vpn-backend/internal/repository"
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

func (p *PaymentService) HasAccess(user *models.User) bool {
	if user.IsBanned {
		return false
	}
	// Триал активен
	if time.Now().Before(user.TrialEndsAt) {
		return true
	}
	// Подписка активна
	if user.TariffExpiresAt.After(time.Now()) {
		return true
	}
	return false
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

	// Access decision is based on subscription expiry/trial only (date-based).
	// We no longer enforce traffic-based cutoffs here — return whether user currently has access.
	return p.HasAccess(user), nil
}

func (p *PaymentService) CheckTariffLimits(userID int) (bool, error) {
	user, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	// For now, treat access as date-based: if user has an active trial or tariff by date, return true.
	return p.HasAccess(user), nil
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

type yooCreatePaymentRequest struct {
	Amount struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"amount"`
	Confirmation struct {
		Type      string `json:"type"`
		ReturnURL string `json:"return_url"`
	} `json:"confirmation"`
	Capture     bool              `json:"capture"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type yooCreatePaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		Type string `json:"type"`
		URL  string `json:"confirmation_url"`
	} `json:"confirmation"`
}

func (p *PaymentService) CreateYooKassaPayment(userID int, months int, returnURL string, shopID string, secret string, amountRub int, description string) (string, string, error) {
	// Создать запись платежа (pending)
	payment := &models.Payment{
		UserID:        userID,
		Amount:        amountRub,
		TariffID:      0,
		PaymentMethod: "yookassa",
		Status:        "pending",
		Provider:      "yookassa",
		CreatedAt:     time.Now(),
	}
	if err := p.UserRepo.CreatePayment(payment); err != nil {
		return "", "", fmt.Errorf("failed to create payment record: %w", err)
	}

	req := yooCreatePaymentRequest{}
	req.Amount.Value = fmt.Sprintf("%d.00", amountRub)
	req.Amount.Currency = "RUB"
	req.Confirmation.Type = "redirect"
	req.Confirmation.ReturnURL = returnURL
	req.Capture = true
	req.Description = description
	req.Metadata = map[string]string{
		"user_id": fmt.Sprintf("%d", userID),
		"months":  fmt.Sprintf("%d", months),
	}
	body, _ := json.Marshal(req)

	apiReq, _ := http.NewRequest("POST", "https://api.yookassa.ru/v3/payments", bytes.NewReader(body))
	apiReq.Header.Set("Content-Type", "application/json")
	// Idempotence-Key
	apiReq.Header.Set("Idempotence-Key", fmt.Sprintf("%d-%d", userID, time.Now().UnixNano()))
	basic := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", shopID, secret)))
	apiReq.Header.Set("Authorization", "Basic "+basic)

	resp, err := http.DefaultClient.Do(apiReq)
	if err != nil {
		return "", "", fmt.Errorf("yookassa request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("yookassa returned status %d", resp.StatusCode)
	}
	var yres yooCreatePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&yres); err != nil {
		return "", "", fmt.Errorf("failed to decode yookassa response: %w", err)
	}
	// Обновить Payment.ProviderID
	_ = p.UserRepo.UpdatePaymentProviderID(payment.UserID, fmt.Sprint(payment.ID), yres.ID)

	return yres.Confirmation.URL, yres.ID, nil
}

// OnYooKassaWebhookSucceeded обновляет подписку пользователя на указанный период
func (p *PaymentService) OnYooKassaWebhookSucceeded(userID int, months int, providerID string, amount int) error {
	user, err := p.UserRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	base := time.Now()
	if user.TariffExpiresAt.After(base) {
		base = user.TariffExpiresAt
	}
	newExpiry := base.AddDate(0, months, 0)
	if err := p.UserRepo.UpdateTariffExpiry(userID, newExpiry); err != nil {
		return fmt.Errorf("failed to update tariff expiry: %w", err)
	}

	// Send notifications: to user (if linked) and to admins via bot (if available)
	// We reference botService and sendToAdmins which live in the same package `services`.
	if botService != nil {
		// Notify user in their Telegram chat if telegram_id is set
		if user.TelegramID != 0 {
			receipt := fmt.Sprintf("Платёж подтверждён. Сумма: %d ₽. Период: %d мес. Провайдер ID: %s. Подписка продлена до: %s",
				amount, months, providerID, newExpiry.Format("2006-01-02 15:04:05"))
			sendMessage(user.TelegramID, receipt)
		}

		// Notify admins
		adminMsg := fmt.Sprintf("Платёж успешно завершён пользователем %s (id=%d). Сумма: %d ₽. Месяцев: %d. ProviderID: %s. Подписка до: %s",
			user.Email, user.ID, amount, months, providerID, newExpiry.Format("2006-01-02 15:04:05"))
		sendToAdmins(adminMsg)
	}

	return nil
}
