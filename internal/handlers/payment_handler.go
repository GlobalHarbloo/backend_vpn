package handlers

import (
	"encoding/json"
	"net/http"
	"vpn-backend/internal/middleware"
	"vpn-backend/internal/services"
	"vpn-backend/internal/utils"

	"github.com/gorilla/mux"
)

type PaymentHandler struct {
	PaymentService *services.PaymentService
}

func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{PaymentService: paymentService}
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var data struct {
		Amount        int    `json:"amount"`
		TariffID      int    `json:"tariff_id"`
		PaymentMethod string `json:"payment_method"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.PaymentService.CreatePayment(userID, data.Amount, data.TariffID, data.PaymentMethod)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create payment")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{"status": "payment created"})
}

func (h *PaymentHandler) GetUserPayments(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	payments, err := h.PaymentService.GetPaymentsByUserID(userID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get payments")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, payments)
}

func (h *PaymentHandler) GetPaymentByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	paymentID := mux.Vars(r)["id"]

	payment, err := h.PaymentService.GetPaymentByID(userID, paymentID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Payment not found")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) UpdatePaymentStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	paymentID := mux.Vars(r)["id"]

	var data struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.PaymentService.UpdatePaymentStatus(userID, paymentID, data.Status)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update payment status")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"status": "payment status updated"})
}
