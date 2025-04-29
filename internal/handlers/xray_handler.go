package handlers

import (
	"fmt"
	"net/http"
	"vpn-backend/internal/services"
	"vpn-backend/internal/utils"
)

type XrayHandler struct {
	Service *services.XrayService
}

func NewXrayHandler(s *services.XrayService) *XrayHandler {
	return &XrayHandler{Service: s}
}

func (h *XrayHandler) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	if err := h.Service.RegenerateConfig(); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Config reload failed: %v", err))
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "config reloaded"})
}

func (h *XrayHandler) Restart(w http.ResponseWriter, r *http.Request) {
	if err := h.Service.RestartXray(); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Restart failed: %v", err))
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "xray restarted"})
}
