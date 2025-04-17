package handlers

import (
	"net/http"
	"vpn-backend/internal/services"
)

type XrayHandler struct {
	Service *services.XrayService
}

func NewXrayHandler(service *services.XrayService) *XrayHandler {
	return &XrayHandler{Service: service}
}

func (h *XrayHandler) Restart(w http.ResponseWriter, r *http.Request) {
	err := h.Service.RestartXray()
	if err != nil {
		http.Error(w, "Ошибка перезапуска Xray", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Xray перезапущен"))
}

func (h *XrayHandler) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	err := h.Service.RegenerateConfig()
	if err != nil {
		http.Error(w, "Ошибка перезагрузки конфигурации Xray", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Конфигурация Xray перезагружена"))
}
