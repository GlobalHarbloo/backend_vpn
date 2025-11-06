package services

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// CreateRobokassaPayment создает запись о платеже и формирует ссылку для оплаты через Robokassa.
// robokassaLogin — логин мерчанта, password1 — пароль для формирования подписи на редирект,
// password2 — пароль для проверки уведомлений (POST).
func (p *PaymentService) CreateRobokassaPayment(userID int, months int, amountRub int, description string, robokassaLogin, password1 string, baseReturnURL string) (string, string, string, string, error) {
	// Создать запись платежа (pending)
	payment := &models.Payment{
		UserID:        userID,
		Amount:        amountRub,
		TariffID:      months, // store months temporarily
		PaymentMethod: "robokassa",
		Status:        "pending",
		Provider:      "robokassa",
		CreatedAt:     time.Now(),
	}
	if err := p.UserRepo.CreatePayment(payment); err != nil {
		return "", "", "", "", fmt.Errorf("failed to create payment record: %w", err)
	}

	// Use payment.ID as InvId
	invID := fmt.Sprintf("%d", payment.ID)
	outSum := fmt.Sprintf("%d", amountRub)

	// Signature: md5(MrchLogin:OutSum:InvId:Password1)
	signSrc := fmt.Sprintf("%s:%s:%s:%s", robokassaLogin, outSum, invID, password1)
	md := md5.Sum([]byte(signSrc))
	signature := hex.EncodeToString(md[:])

	// Build payment URL (classic robokassa redirect)
	// Example: https://auth.robokassa.ru/Merchant/Index.aspx?MrchLogin=login&OutSum=100&InvId=1&Desc=desc&SignatureValue=sign
	payURL := fmt.Sprintf("https://auth.robokassa.ru/Merchant/Index.aspx?MrchLogin=%s&OutSum=%s&InvId=%s&Desc=%s&SignatureValue=%s",
		urlQueryEscape(robokassaLogin), urlQueryEscape(outSum), urlQueryEscape(invID), urlQueryEscape(description), urlQueryEscape(signature))

	// Build simple success/fail URLs for merchant return (frontend or server)
	successURL := ""
	failURL := ""
	if baseReturnURL != "" {
		base := strings.TrimRight(baseReturnURL, "/")
		successURL = fmt.Sprintf("%s/payments/robokassa/success?inv=%s", base, invID)
		failURL = fmt.Sprintf("%s/payments/robokassa/fail?inv=%s", base, invID)
	}

	return payURL, invID, successURL, failURL, nil
}

// HandleRobokassaCallback processes server-to-server notifications from Robokassa.
// It validates signature and, if payment is valid, marks it paid and extends subscription.
func (p *PaymentService) HandleRobokassaCallback(form map[string][]string, password2 string) error {
	// Robokassa typically sends OutSum, InvId and SignatureValue (case-insensitive)
	outSum := firstFormVal(form, "OutSum")
	invId := firstFormVal(form, "InvId")
	signature := firstFormVal(form, "SignatureValue")
	if outSum == "" || invId == "" || signature == "" {
		return fmt.Errorf("missing required robokassa fields")
	}

	// signature check: md5(OutSum:InvId:Password2)
	signSrc := fmt.Sprintf("%s:%s:%s", outSum, invId, password2)
	md := md5.Sum([]byte(signSrc))
	expected := strings.ToLower(hex.EncodeToString(md[:]))
	if strings.ToLower(signature) != expected {
		return fmt.Errorf("invalid signature")
	}

	// Find payment by invId
	payment, err := p.UserRepo.GetPaymentByIDAny(invId)
	if err != nil {
		return fmt.Errorf("payment not found: %w", err)
	}

	// Parse OutSum (Robokassa may send decimals like 100.00). Compare with stored integer amount (RUB whole units).
	// We accept small formatting differences but require the integer rubles to match.
	outSumF, perr := strconv.ParseFloat(outSum, 64)
	if perr != nil {
		return fmt.Errorf("invalid OutSum value: %w", perr)
	}
	// Convert to int rubles by rounding to nearest integer (robokassa usually sends .00)
	outSumInt := int(outSumF + 0.5)
	if outSumInt != payment.Amount {
		return fmt.Errorf("amount mismatch: notification %d vs payment record %d", outSumInt, payment.Amount)
	}

	// Idempotency: if already paid, treat as success
	if strings.ToLower(payment.Status) == "paid" {
		return nil
	}

	// Mark payment as paid
	if err := p.UserRepo.UpdatePaymentStatus(payment.UserID, invId, "paid"); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Determine months from TariffID field stored earlier
	months := payment.TariffID
	amount := payment.Amount

	// Extend subscription
	if err := p.OnYooKassaWebhookSucceeded(payment.UserID, months, invId, amount); err != nil {
		return fmt.Errorf("failed to apply subscription: %w", err)
	}

	return nil
}

// helpers
func firstFormVal(form map[string][]string, key string) string {
	if v, ok := form[key]; ok && len(v) > 0 {
		return v[0]
	}
	// try lowercase
	if v, ok := form[strings.ToLower(key)]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// urlQueryEscape is a tiny helper to escape query values
func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
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
