package handlers

import (
	"net/http"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/services"
	"vpn-backend/internal/utils"
)

type ConfigHandler struct {
	Xray *services.XrayService
}

func NewConfigHandler(x *services.XrayService) *ConfigHandler {
	return &ConfigHandler{Xray: x}
}

func (h *ConfigHandler) RegenerateConfig(w http.ResponseWriter, r *http.Request) {
	if err := h.Xray.RegenerateConfig(); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to regenerate config: "+err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "config regenerated"})
}

func (h *ConfigHandler) RestartXray(w http.ResponseWriter, r *http.Request) {
	if err := h.Xray.RestartXray(); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to restart Xray: "+err.Error())
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "xray restarted"})
}

func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.Xray.Repo.FindByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	config, err := h.Xray.GenerateUserConfig(user)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate config")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(config)
}
