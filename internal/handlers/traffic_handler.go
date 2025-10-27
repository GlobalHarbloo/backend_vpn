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
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"traffic_bytes": int64(0), "traffic_human": "0 B"})
		return
	}

	traffic, err := h.Traffic.GetUserTraffic(user.UUID)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"traffic_bytes": int64(0), "traffic_human": "0 B"})
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"traffic_bytes": traffic, "traffic_human": utils.BytesToHuman(traffic)})
}
