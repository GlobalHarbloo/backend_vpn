package handlers

import (
	"net/http"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/services"
	"vpn-backend/internal/utils"
)

type TrafficHandler struct {
	Traffic *services.TrafficService
}

func NewTrafficHandler(traffic *services.TrafficService) *TrafficHandler {
	return &TrafficHandler{Traffic: traffic}
}

func (h *TrafficHandler) GetTraffic(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.Traffic.UserRepo.FindByID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	traffic, err := h.Traffic.GetUserTraffic(user.UUID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get traffic")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]int64{"traffic": traffic})
}
