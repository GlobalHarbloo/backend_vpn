package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/yourusername/vpn-backend/internal/middleware"
	"github.com/yourusername/vpn-backend/internal/services"
	"github.com/yourusername/vpn-backend/internal/utils"

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

// CreateYooKassaPaymentHandler создаёт платёж и возвращает ссылку для оплаты
func (h *PaymentHandler) CreateYooKassaPaymentHandler(cfgReturnURL, cfgShopID, cfgSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r)
		if !ok {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		var data struct {
			Months int `json:"months"`
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil || (data.Months != 1 && data.Months != 3) {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid months")
			return
		}
		amount := 200
		if data.Months == 3 {
			amount = 500
		}
		url, providerID, err := h.PaymentService.CreateYooKassaPayment(userID, data.Months, cfgReturnURL, cfgShopID, cfgSecret, amount, "VPN subscription")
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create payment")
			return
		}
		utils.RespondWithJSON(w, http.StatusCreated, map[string]string{"confirmation_url": url, "provider_id": providerID})
	}
}

// CreateRobokassaPaymentHandler создаёт платёж в Robokassa и возвращает ссылку для оплаты.
func (h *PaymentHandler) CreateRobokassaPaymentHandler(robokassaLogin, password1, baseReturnURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r)
		if !ok {
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		var data struct {
			Months int `json:"months"`
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil || (data.Months != 1 && data.Months != 3) {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid months")
			return
		}
		amount := 299
		if data.Months == 3 {
			amount = 749
		}
		url, invId, successURL, failURL, err := h.PaymentService.CreateRobokassaPayment(userID, data.Months, amount, "VPN subscription", robokassaLogin, password1, baseReturnURL)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create robokassa payment")
			return
		}
		utils.RespondWithJSON(w, http.StatusCreated, map[string]string{"confirmation_url": url, "inv_id": invId, "success_url": successURL, "fail_url": failURL})
	}
}

// RobokassaWebhookHandler принимает уведомления от Robokassa (server-to-server)
func (h *PaymentHandler) RobokassaWebhookHandler(password2 string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Robokassa may send either POST form or GET query. Parse both.
		if err := r.ParseForm(); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid form")
			return
		}
		if err := h.PaymentService.HandleRobokassaCallback(r.Form, password2); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Robokassa callback error: %v", err))
			return
		}
		// Robokassa expects plain OK response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}

// YooKassaWebhookHandler принимает вебхуки от ЮKassa
func (h *PaymentHandler) YooKassaWebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var event struct {
			Event  string `json:"event"`
			Object struct {
				ID       string            `json:"id"`
				Status   string            `json:"status"`
				Metadata map[string]string `json:"metadata"`
				Amount   struct {
					Value string `json:"value"`
				} `json:"amount"`
			} `json:"object"`
		}
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid body")
			return
		}
		// Ожидаем, что метаданные содержат user_id и months (если настроим в будущем)
		userIDStr := event.Object.Metadata["user_id"]
		monthsStr := event.Object.Metadata["months"]
		if event.Event == "payment.succeeded" && event.Object.Status == "succeeded" && userIDStr != "" && monthsStr != "" {
			uid, _ := strconv.Atoi(userIDStr)
			months, _ := strconv.Atoi(monthsStr)
			providerID := event.Object.ID
			// try to parse amount if available (Amount.Value is already a string)
			amount := 0
			val := event.Object.Amount.Value
			if val != "" {
				// value like "200.00" — parse integer rubles (take part before dot)
				parts := strings.SplitN(val, ".", 2)
				if iv, err := strconv.Atoi(parts[0]); err == nil {
					amount = iv
				}
			}
			_ = h.PaymentService.OnYooKassaWebhookSucceeded(uid, months, providerID, amount)
		}
		w.WriteHeader(http.StatusOK)
	}
}
