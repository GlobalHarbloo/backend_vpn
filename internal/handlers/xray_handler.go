package handlers

import (
	"encoding/json"
	"net/http"
	"vpn-backend/internal/services"
)

type XrayHandler struct {
	service *services.XrayService
}

func NewXrayHandler(service *services.XrayService) *XrayHandler {
	return &XrayHandler{service: service}
}

func (h *XrayHandler) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.Repo().GetAllUsers()
	if err != nil {
		http.Error(w, "Ошибка получения пользователей", http.StatusInternalServerError)
		return
	}
	err = h.service.RegenerateConfig(users)
	if err != nil {
		http.Error(w, "Ошибка генерации конфига", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "config updated"})
}

func (h *XrayHandler) Restart(w http.ResponseWriter, r *http.Request) {
	err := h.service.RestartXray()
	if err != nil {
		http.Error(w, "Ошибка перезапуска Xray", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "xray restarted"})
}
