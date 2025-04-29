package handlers

import (
	"encoding/json"
	"net/http"
	"vpn-backend/internal/services"
	"vpn-backend/internal/utils"

	"github.com/google/uuid"
)

type AuthHandler struct {
	Auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{Auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	uuidStr := uuid.New().String()
	const baseTariffID = 1 // Lite

	user, err := h.Auth.Register(data.Email, data.Password, uuidStr, baseTariffID)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
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
