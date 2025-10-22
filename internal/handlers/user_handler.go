package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"github.com/yourusername/vpn-backend/config"
	"github.com/yourusername/vpn-backend/internal/middleware"
	"github.com/yourusername/vpn-backend/internal/services"
	"github.com/yourusername/vpn-backend/internal/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	Auth    *services.AuthService
	Payment *services.PaymentService
	Xray    *services.XrayService
	Traffic *services.TrafficService
}

func NewUserHandler(auth *services.AuthService, payment *services.PaymentService, xray *services.XrayService, traffic *services.TrafficService) *UserHandler {
	return &UserHandler{
		Auth:    auth,
		Payment: payment,
		Xray:    xray,
		Traffic: traffic,
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
	_ = h.Auth.UserRepo.UpdateUsedTraffic(int(user.ID), baseTraffic)

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

	// Автоматический вход: выдаем JWT токен сразу после успешной регистрации
	token, err := h.Auth.GenerateJWT(int(user.ID))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user":  user,
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

	var token string
	var err error

	if data.TelegramID != 0 {
		token, err = h.Auth.AuthenticateByTelegramID(data.TelegramID)
	} else {
		token, err = h.Auth.AuthenticateUser(data.Email, data.Password)
	}

	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"token": token})
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

	resp := struct {
		ID          int       `json:"id"`
		Email       string    `json:"email"`
		UUID        string    `json:"uuid"`
		TariffID    int       `json:"tariff_id"`
		Traffic     int64     `json:"traffic"`
		ExpiresAt   time.Time `json:"expires_at"`
		TrialEndsAt time.Time `json:"trial_ends_at"`
		HasAccess   bool      `json:"has_access"`
	}{
		ID:          int(user.ID),
		Email:       user.Email,
		UUID:        user.UUID,
		TariffID:    user.TariffID,
		Traffic:     traffic,
		ExpiresAt:   expiry,
		TrialEndsAt: user.TrialEndsAt,
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

	// Сначала удаляем из Xray конфига
	if err := h.Xray.RemoveUserFromConfig(user.UUID); err != nil {
		log.Printf("[Xray] Error removing user from config: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update Xray config")
		return
	}

	// Перезапуск Xray после удаления пользователя
	h.Xray.ScheduleRestart()

	// Удаляем пользователя из базы данных
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
	token := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)
	_ = h.Auth.UserRepo.SetPasswordResetToken(data.Email, token, expiresAt)
	_ = h.Auth.UserRepo.UpdatePasswordResetRequestedAt(data.Email, time.Now())
	// Получаем URL фронтенда из конфигурации
	cfg := config.Load()
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", cfg.FrontendURL, token)
	if err := utils.SendResetEmail(user.Email, resetLink); err != nil {
		log.Printf("[EMAIL] Ошибка отправки письма: %v", err)
	}
	log.Printf("[RESET] Сброс пароля отправлен на %s", user.Email)
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "If email exists, reset link sent"})
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	user, err := h.Auth.UserRepo.FindByPasswordResetToken(data.Token)
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
	user, err := h.Auth.UserRepo.FindByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(data.OldPassword)); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Wrong old password")
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), bcrypt.DefaultCost)
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
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "password changed"})
}
