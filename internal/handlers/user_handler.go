package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yourusername/vpn-backend/internal/middleware"
	"github.com/yourusername/vpn-backend/internal/services"
	"github.com/yourusername/vpn-backend/internal/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	Auth        *services.AuthService
	Payment     *services.PaymentService
	Xray        *services.XrayService
	Traffic     *services.TrafficService
	BotUsername string
}

func NewUserHandler(auth *services.AuthService, payment *services.PaymentService, xray *services.XrayService, traffic *services.TrafficService, botUsername string) *UserHandler {
	return &UserHandler{
		Auth:        auth,
		Payment:     payment,
		Xray:        xray,
		Traffic:     traffic,
		BotUsername: botUsername,
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	uuidStr := uuid.New().String()
	const baseTariffID = 1       // Базовый тариф
	const baseTraffic = 10485760 // 10 МБ в байтах

	user, err := h.Auth.Register(data.Email, data.Password, uuidStr, baseTariffID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Registration failed: %v", err))
		return
	}

	// Установить базовый тариф и трафик для нового пользователя
	_ = h.Auth.UserRepo.UpdateUserTariff(int(user.ID), baseTariffID)
	// Ensure used_traffic is non-null in DB: set to baseTraffic (10MB)
	if err := h.Auth.UserRepo.UpdateUsedTraffic(int(user.ID), baseTraffic); err != nil {
		log.Printf("[UserHandler] Failed to set initial used traffic for user %d: %v", int(user.ID), err)
	}

	// Добавление пользователя в конфигурацию Xray
	if err := h.Xray.AddUserToConfig(user); err != nil {
		log.Printf("[Xray] Error adding user to config: %v", err)
		// Откатываем создание пользователя, чтобы не оставлять несогласованное состояние
		_ = h.Auth.UserRepo.Delete(int(user.ID))
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update Xray config")
		return
	}

	// Проверка, что пользователь есть в clients Xray config
	if ok, err := h.Xray.CheckUserInConfig(user.UUID); err != nil {
		log.Printf("[Xray] Error checking user in config: %v", err)
	} else if !ok {
		log.Printf("[Xray] User %s NOT found in Xray config after registration", user.UUID)
	} else {
		log.Printf("[Xray] User %s successfully added to Xray config", user.UUID)
	}

	// Перезапуск Xray
	h.Xray.ScheduleRestart()

	// Автоматический вход: выдаем access и refresh токены сразу после успешной регистрации
	access, refresh, err := h.Auth.CreateTokens(int(user.ID))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"token":         access,
		"access_token":  access,
		"refresh_token": refresh,
		"user":          user,
	})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		TelegramID int64  `json:"telegram_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var access, refresh string
	var err error

	if data.TelegramID != 0 {
		access, refresh, err = h.Auth.AuthenticateByTelegramID(data.TelegramID)
	} else {
		access, refresh, err = h.Auth.AuthenticateUser(data.Email, data.Password)
	}

	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"token": access, "access_token": access, "refresh_token": refresh})
}

func (h *UserHandler) ChangeTariff(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var data struct {
		TariffID int `json:"tariff_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.Payment.ChangeTariff(userID, data.TariffID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to change tariff")
		return
	}

	user, err := h.Auth.UserRepo.FindByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to find user")
		return
	}

	if err := h.Xray.UpdateUserTariff(user.UUID, data.TariffID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update Xray config")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "tariff changed"})
}

func (h *UserHandler) UpgradeTariff(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var data struct {
		TariffID int `json:"tariff_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.Payment.ChangeTariff(userID, data.TariffID); err != nil {
		http.Error(w, "Failed to change tariff", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.Auth.UserRepo.FindByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	traffic, err := h.Traffic.GetUserTraffic(user.UUID)
	if err != nil {
		// Если не удалось получить трафик — возвращаем 0 вместо ошибки
		traffic = 0
	}

	expiry, err := h.Payment.GetTariffExpiry(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get tariff expiry")
		return
	}

	hasAccess := h.Payment.HasAccess(user)

	// Use pointer times so zero value is marshalled as null instead of 0001-01-01
	var expiryPtr *time.Time
	if !expiry.IsZero() {
		expiryPtr = &expiry
	}
	var trialPtr *time.Time
	if !user.TrialEndsAt.IsZero() {
		trialPtr = &user.TrialEndsAt
	}

	resp := struct {
		ID          int        `json:"id"`
		Email       string     `json:"email"`
		UUID        string     `json:"uuid"`
		TariffID    int        `json:"tariff_id"`
		Traffic     int64      `json:"traffic"`
		ExpiresAt   *time.Time `json:"expires_at"`
		TrialEndsAt *time.Time `json:"trial_ends_at"`
		HasAccess   bool       `json:"has_access"`
	}{
		ID:          int(user.ID),
		Email:       user.Email,
		UUID:        user.UUID,
		TariffID:    user.TariffID,
		Traffic:     traffic,
		ExpiresAt:   expiryPtr,
		TrialEndsAt: trialPtr,
		HasAccess:   hasAccess,
	}

	utils.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.Auth.UserRepo.FindByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Attempt to remove user from Xray config and restart Xray before deleting from DB.
	// This ensures the running Xray instance is consistent. If restart fails,
	// we attempt to restore the previous config and abort deletion.
	if err := h.Xray.RemoveUserFromConfig(user.UUID); err != nil {
		log.Printf("[Xray] Error removing user from config: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update Xray config")
		return
	}

	// Try to restart Xray synchronously and ensure it came up.
	if err := h.Xray.RestartXray(); err != nil {
		log.Printf("[Xray] Restart failed after removing user %s: %v", user.UUID, err)
		// Try to rollback config from backup
		if rbErr := h.Xray.RestoreBackup(); rbErr != nil {
			log.Printf("[Xray] Failed to restore backup after restart failure: %v", rbErr)
		} else {
			// attempt to restart back to the previous config
			if rrErr := h.Xray.RestartXray(); rrErr != nil {
				log.Printf("[Xray] Restart after restoring backup also failed: %v", rrErr)
			}
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Xray restart failed after config update; aborting deletion")
		return
	}

	// Now safe to delete user from DB
	if err := h.Auth.UserRepo.Delete(userID); err != nil {
		log.Printf("[DB] Error deleting user: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "account deleted"})
}

// Новый endpoint для запроса сброса пароля (заглушка)
func (h *UserHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	user, err := h.Auth.UserRepo.GetUserByEmail(data.Email)
	if err != nil {
		log.Printf("[RESET] Запрос сброса для несуществующего email: %s", data.Email)
		utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "If email exists, reset link sent"})
		return
	}
	// Антиспам: не более 5 запросов за 30 минут
	limit := 5
	window := 30 * time.Minute
	count, _ := h.Auth.UserRepo.CountPasswordResetRequests(data.Email, time.Now().Add(-window))
	if count >= int64(limit) {
		log.Printf("[RESET] Превышен лимит сброса для %s", data.Email)
		utils.RespondWithError(w, http.StatusTooManyRequests, "Слишком много запросов на сброс пароля. Попробуйте позже.")
		return
	}
	// Generate a numeric code (6 digits), store SHA256(hash) in DB, and email the code.
	code, err := generateNumericCode(6)
	if err != nil {
		log.Printf("[RESET] failed to generate code: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate reset code")
		return
	}
	expiresAt := time.Now().Add(15 * time.Minute)
	hashed := fmt.Sprintf("%x", sha256.Sum256([]byte(code)))
	if err := h.Auth.UserRepo.SetPasswordResetToken(data.Email, hashed, expiresAt); err != nil {
		log.Printf("[RESET] failed to store reset token: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate reset code")
		return
	}
	_ = h.Auth.UserRepo.UpdatePasswordResetRequestedAt(data.Email, time.Now())
	// Send numeric code via email
	if err := utils.SendResetCodeEmail(user.Email, code, expiresAt); err != nil {
		log.Printf("[EMAIL] Ошибка отправки кода: %v", err)
	}
	log.Printf("[RESET] Код сброса пароля отправлен на %s", user.Email)
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "If email exists, reset code sent"})
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	user, err := h.Auth.UserRepo.FindByPasswordResetToken(data.Code)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid or expired token")
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}
	user.Password = string(hashedPassword)
	user.PasswordResetToken = ""
	user.PasswordResetExpiresAt = time.Time{}
	if err := h.Auth.UserRepo.DB.Save(user).Error; err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "password reset successful"})
}

// generateNumericCode generates a cryptographically secure numeric code with n digits.
func generateNumericCode(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("invalid code length")
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) // 10^n
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	s := num.Text(10)
	if len(s) < n {
		s = strings.Repeat("0", n-len(s)) + s
	}
	return s, nil
}

func (h *UserHandler) CheckToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]int{"user_id": userID})
}

func (h *UserHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := h.Auth.UserRepo.FindByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	// Формируем строку подписки для пользователя
	sub := fmt.Sprintf(
		"vless://%s@193.124.182.210:10000?encryption=none&security=tls&type=ws&path=%%2F#VPNClient",
		user.UUID,
	)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(sub))
}

// GetBotLink returns a Telegram deep link that contains a short-lived token for automatic user recognition in the bot.
func (h *UserHandler) GetBotLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if h.BotUsername == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Telegram bot not configured on server")
		return
	}

	// Create a short-lived token (15 minutes)
	token, err := h.Auth.GenerateBotToken(userID, 15*time.Minute)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create bot token")
		return
	}

	// Provide both an HTTPS link and a tg:// resolve link. The app can prefer the
	// tg:// scheme to open the native Telegram app when available.
	httpsLink := fmt.Sprintf("https://t.me/%s?start=%s", h.BotUsername, url.QueryEscape(token))
	tgLink := fmt.Sprintf("tg://resolve?domain=%s&start=%s", h.BotUsername, url.QueryEscape(token))
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"link": httpsLink, "tg_link": tgLink})
}

func (h *UserHandler) GetHiddifyConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := h.Auth.UserRepo.FindByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	// Генерируем полноценный YAML-конфиг для Clash/Hiddify Core (VLESS через WS+TLS)
	yaml := fmt.Sprintf(`
port: 7890
socks-port: 7891
allow-lan: true
mode: rule
log-level: info
external-controller: 127.0.0.1:9090

proxies:
  - name: "%s"
    type: vless
    server: 193.124.182.210
    port: 10000
    uuid: %s
    network: ws
    tls: true
    ws-opts:
      path: /

proxy-groups:
  - name: "auto"
    type: select
    proxies:
      - "%s"

rules:
  - MATCH,auto

tun:
  enable: true
  stack: system
  auto-route: true
  auto-detect-interface: true
`, user.Email, user.UUID, user.Email)
	w.Header().Set("Content-Type", "text/yaml")
	w.Write([]byte(yaml))
}

func GenerateSubscriptionFile(configPath, outputPath string) error {
	// Структуры для парсинга только нужных частей
	type Client struct {
		Email string `json:"email"`
		ID    string `json:"id"`
	}
	type Inbound struct {
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
		Settings struct {
			Clients []Client `json:"clients"`
		} `json:"settings"`
		StreamSettings struct {
			Network    string `json:"network"`
			Security   string `json:"security"`
			WsSettings struct {
				Path string `json:"path"`
			} `json:"wsSettings"`
		} `json:"streamSettings"`
	}
	type Config struct {
		Inbounds []Inbound `json:"inbounds"`
	}

	// Читаем config.json
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Ищем inbound с protocol == "vless"
	var vlessInbound *Inbound
	for _, inb := range cfg.Inbounds {
		if inb.Protocol == "vless" {
			vlessInbound = &inb
			break
		}
	}
	if vlessInbound == nil {
		return fmt.Errorf("no vless inbound found")
	}

	// Формируем строки подписки
	host := "193.124.182.210"
	port := vlessInbound.Port
	network := vlessInbound.StreamSettings.Network
	security := vlessInbound.StreamSettings.Security
	path := vlessInbound.StreamSettings.WsSettings.Path
	if path == "" {
		path = "/"
	}
	subLines := ""
	for _, client := range vlessInbound.Settings.Clients {
		subLines += fmt.Sprintf(
			"vless://%s@%s:%d?encryption=none&security=%s&type=%s&path=%%2F#%s\n",
			client.ID, host, port, security, network, client.Email,
		)
	}

	// Записываем в subscription.txt
	return ioutil.WriteFile(outputPath, []byte(subLines), 0644)
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var data struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	// Delegate password verification and update to repository helper to keep logic consistent
	if err := h.Auth.UserRepo.UpdatePasswordIfMatchesUserID(userID, data.OldPassword, data.NewPassword); err != nil {
		// Reuse error messages from repo, but avoid leaking details
		utils.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to change password: %v", err))
		return
	}
	// Password successfully changed
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}

// RefreshToken accepts a refresh token and returns new access and refresh tokens (rotates refresh token).
func (h *UserHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var data struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if data.RefreshToken == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	user, err := h.Auth.UserRepo.FindByRefreshToken(data.RefreshToken)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	access, refresh, err := h.Auth.CreateTokens(int(user.ID))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create tokens")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"access_token": access, "refresh_token": refresh})
}

// Logout clears server-side refresh token for the authenticated user.
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err := h.Auth.UserRepo.ClearRefreshToken(userID); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to logout")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}
