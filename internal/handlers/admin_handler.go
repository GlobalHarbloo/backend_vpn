package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"vpn-backend/internal/repository"
	"vpn-backend/internal/utils"

	"github.com/gorilla/mux"
)

type AdminHandler struct {
	Repo *repository.UserRepository
}

func NewAdminHandler(repo *repository.UserRepository) *AdminHandler {
	return &AdminHandler{Repo: repo}
}

func (h *AdminHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Repo.GetAllUsers()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error fetching users")
		return
	}
	utils.RespondWithJSON(w, http.StatusOK, users)
}

// Request body: { "ban": true }
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var body struct {
		Ban bool `json:"ban"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = h.Repo.BanUser(id, body.Ban)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update user ban status")
		return
	}
	w.WriteHeader(http.StatusOK)
}
