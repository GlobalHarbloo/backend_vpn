package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/vpn-backend/internal/services"
	"github.com/yourusername/vpn-backend/internal/utils"

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

	// Create tokens for the new user
	access, refresh, err := h.Auth.CreateTokens(int(user.ID))
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create tokens")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{"token": access, "access_token": access, "refresh_token": refresh, "user": user})
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
