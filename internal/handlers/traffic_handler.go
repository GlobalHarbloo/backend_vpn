package handlers

import (
	"net/http"
	"github.com/yourusername/vpn-backend/internal/middleware"
	"github.com/yourusername/vpn-backend/internal/services"
	"github.com/yourusername/vpn-backend/internal/utils"
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
		utils.RespondWithJSON(w, http.StatusOK, map[string]int64{"traffic": 0})
		return
	}

	traffic, err := h.Traffic.GetUserTraffic(user.UUID)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusOK, map[string]int64{"traffic": 0})
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]int64{"traffic": traffic})
}
