package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/services"
	"vpn-backend/internal/utils"

	"github.com/google/uuid"
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
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update Xray config")
		return
	}

	// Перезапуск Xray
	h.Xray.ScheduleRestart()

	// --- ДОБАВЛЕНИЕ В ОБЩИЙ subscription.txt ---
	subLine := fmt.Sprintf(
		"vless://%s@193.124.182.210:10000?encryption=none&security=tls&type=ws&path=%%2F#VPNClient\n",
		user.UUID,
	)
	subFilePath := "/path/to/subscription.txt" // Укажите тот же путь, что и в main.go
	f, err := os.OpenFile(subFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = f.WriteString(subLine)
		f.Close()
	}
	// --- КОНЕЦ ДОБАВЛЕНИЯ ---

	utils.RespondWithJSON(w, http.StatusCreated, user)
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

	resp := struct {
		ID        int       `json:"id"`
		Email     string    `json:"email"`
		UUID      string    `json:"uuid"`
		TariffID  int       `json:"tariff_id"`
		Traffic   int64     `json:"traffic"`
		ExpiresAt time.Time `json:"expires_at"`
	}{
		ID:        int(user.ID),
		Email:     user.Email,
		UUID:      user.UUID,
		TariffID:  user.TariffID,
		Traffic:   traffic,
		ExpiresAt: expiry,
	}

	utils.RespondWithJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tx := h.Auth.UserRepo.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	user, err := h.Auth.UserRepo.FindByID(userID)
	if err != nil {
		tx.Rollback()
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	if err := h.Xray.RemoveUserFromConfig(user.UUID); err != nil {
		tx.Rollback()
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update Xray config")
		return
	}

	if err := h.Auth.UserRepo.Delete(userID); err != nil {
		tx.Rollback()
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	tx.Commit()
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "account deleted"})
}

// New method to handle password reset request
func (h *UserHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Implement password reset request logic (e.g., send email with reset link)
	// For now, just return a success message
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "password reset initiated"})
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
