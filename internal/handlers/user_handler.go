package handlers

import (
	"encoding/json"
	"net/http"
	"vpn-backend/internal/models"
	"vpn-backend/internal/services"
	"vpn-backend/internal/utils"
)

type UserHandler struct {
	Auth    *services.AuthService
	Payment *services.PaymentService
}

func NewUserHandler(auth *services.AuthService, pay *services.PaymentService) *UserHandler {
	return &UserHandler{Auth: auth, Payment: pay}
}

// ✅ Метод регистрации
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}
	if err := h.Auth.RegisterUser(&user); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Ошибка регистрации")
		return
	}
	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{
		"message": "Пользователь зарегистрирован",
	})
}

// ✅ Метод входа
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds models.User
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}
	token, err := h.Auth.AuthenticateUser(creds.Email, creds.Password)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "Неверные данные")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

// ✅ Метод получения информации о себе
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	userID, err := services.ParseJWT(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"id": userID,
	})
}

// ✅ Метод смены тарифа
func (h *UserHandler) ChangeTariff(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	userID, err := services.ParseJWT(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var data struct {
		TariffID int `json:"tariff_id"`
	}
	json.NewDecoder(r.Body).Decode(&data)
	err = h.Payment.ChangeTariff(userID, data.TariffID)
	if err != nil {
		http.Error(w, "Failed to change tariff", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
