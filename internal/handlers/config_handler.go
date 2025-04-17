package handlers

import (
	"net/http"
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
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to regenerate config")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "config regenerated"})
}

func (h *ConfigHandler) RestartXray(w http.ResponseWriter, r *http.Request) {
	if err := h.Xray.RestartXray(); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to restart Xray")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "xray restarted"})
}
