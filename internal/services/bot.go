package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yourusername/vpn-backend/config"
	"github.com/yourusername/vpn-backend/internal/repository"
)

// Clean, single implementation of an HTTP-polling Telegram bot.
// Keeps implementation minimal: /start <token>, /tariffs, /pay <tariff_id>, /register, /login, /updatepassword.
// Single clean implementation of the Telegram bot (HTTP polling).
type BotService struct {
	Token      string
	Logger     *log.Logger
	UserRepo   *repository.UserRepository
	Auth       *AuthService
	Payment    *PaymentService
	TariffRepo *repository.TariffRepository
	Config     *config.Config
}

var botService *BotService

type conversation struct {
	Action string // "register" or "login"
	Step   int    // 0=expecting email,1=expecting password
	Email  string
}

func InitBot(token string, userRepo *repository.UserRepository, auth *AuthService, payment *PaymentService, tariffRepo *repository.TariffRepository, cfg *config.Config) error {
	logger := log.New(log.Writer(), "bot: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Println("Bot initialized (HTTP polling)")
	botService = &BotService{
		Token:      token,
		Logger:     logger,
		UserRepo:   userRepo,
		Auth:       auth,
		Payment:    payment,
		TariffRepo: tariffRepo,
		Config:     cfg,
	}
	return nil
}

// minimal Telegram types
type tgUpdateResp struct {
	Ok     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}
type tgUpdate struct {
	UpdateID      int              `json:"update_id"`
	Message       *tgMessage       `json:"message,omitempty"`
	CallbackQuery *tgCallbackQuery `json:"callback_query,omitempty"`
}
type tgMessage struct {
	MessageID int     `json:"message_id"`
	From      *tgUser `json:"from,omitempty"`
	Chat      *tgChat `json:"chat,omitempty"`
	Date      int     `json:"date,omitempty"`
	Text      string  `json:"text,omitempty"`
}
type tgUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}
type tgChat struct {
	ID int64 `json:"id"`
}

type tgCallbackQuery struct {
	ID      string     `json:"id"`
	From    *tgUser    `json:"from,omitempty"`
	Message *tgMessage `json:"message,omitempty"`
	Data    string     `json:"data,omitempty"`
}

func sendMessage(chatID int64, text string) {
	if botService == nil {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botService.Token)
	payload := map[string]interface{}{"chat_id": chatID, "text": text}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		botService.Logger.Printf("sendMessage error: %v", err)
		return
	}
	resp.Body.Close()
}

// sendDocumentURL sends a document by URL (Telegram will fetch it).
func sendDocumentURL(chatID int64, fileURL string, caption string) {
	if botService == nil {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botService.Token)
	payload := map[string]interface{}{"chat_id": chatID, "document": fileURL}
	if caption != "" {
		payload["caption"] = caption
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		botService.Logger.Printf("sendDocumentURL error: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// sendToAdmins sends a message to all admin chat IDs configured in the server config.
func sendToAdmins(text string) {
	if botService == nil || botService.Config == nil {
		return
	}
	ids := strings.Split(botService.Config.AdminChatIDs, ",")
	for _, s := range ids {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			botService.Logger.Printf("invalid admin chat id: %s", s)
			continue
		}
		sendMessage(id, text)
	}
}

// answerCallback sends an answerCallbackQuery to Telegram to acknowledge a button press.
func answerCallback(callbackID string, text string) {
	if botService == nil {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", botService.Token)
	payload := map[string]interface{}{"callback_query_id": callbackID, "text": text, "show_alert": false}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		botService.Logger.Printf("answerCallback error: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// sendMessageWithReplyMarkup sends a message with a custom reply_markup (keyboard or inline keyboard)
func sendMessageWithReplyMarkup(chatID int64, text string, replyMarkup interface{}) {
	if botService == nil {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botService.Token)
	payload := map[string]interface{}{"chat_id": chatID, "text": text, "reply_markup": replyMarkup}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		botService.Logger.Printf("sendMessageWithReplyMarkup error: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// helper to send a simple reply keyboard (rows of button texts)
func sendMessageWithKeyboard(chatID int64, text string, keyboard [][]string) {
	// build Telegram reply keyboard format
	rows := make([][]map[string]string, 0, len(keyboard))
	for _, r := range keyboard {
		row := make([]map[string]string, 0, len(r))
		for _, btn := range r {
			row = append(row, map[string]string{"text": btn})
		}
		rows = append(rows, row)
	}
	reply := map[string]interface{}{"keyboard": rows, "resize_keyboard": true, "one_time_keyboard": false}
	sendMessageWithReplyMarkup(chatID, text, reply)
}

func parseCommand(input string) (string, []string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// state helpers backed by DB via UserRepo
func setConversation(chatID int64, c *conversation) {
	if botService == nil || botService.UserRepo == nil {
		return
	}
	_ = botService.UserRepo.SaveBotSession(chatID, c.Action, c.Step, c.Email)
}

func getConversation(chatID int64) *conversation {
	if botService == nil || botService.UserRepo == nil {
		return nil
	}
	s, err := botService.UserRepo.GetBotSession(chatID)
	if err != nil || s == nil {
		return nil
	}
	return &conversation{Action: s.Action, Step: s.Step, Email: s.Email}
}

func clearConversation(chatID int64) {
	if botService == nil || botService.UserRepo == nil {
		return
	}
	_ = botService.UserRepo.DeleteBotSession(chatID)
}

// handleStateMessage returns true if message consumed by conversation state
func handleStateMessage(chatID int64, fromID int64, text string) bool {
	c := getConversation(chatID)
	if c == nil {
		return false
	}
	if c.Action == "register" {
		if c.Step == 0 {
			// expecting email
			c.Email = strings.TrimSpace(text)
			c.Step = 1
			setConversation(chatID, c)
			sendMessage(chatID, t("enter_password"))
			return true
		}
		if c.Step == 1 {
			password := strings.TrimSpace(text)
			// perform registration
			uuidStr := uuid.NewString()
			user, err := botService.Auth.Register(c.Email, password, uuidStr, 1)
			if err != nil {
				sendMessage(chatID, fmt.Sprintf(t("registration_error"), err))
				clearConversation(chatID)
				return true
			}
			// link telegram id
			_ = botService.UserRepo.UpdateUserTelegramID(int(user.ID), fromID)
			sendMessage(chatID, t("registration_success"))
			// offer subscriptions
			payText := t("choose_subscription")
			inlineRows := [][]map[string]interface{}{
				{{"text": t("subscription_1"), "callback_data": "robokassa_buy:1"}},
				{{"text": t("subscription_3"), "callback_data": "robokassa_buy:3"}},
			}
			reply := map[string]interface{}{"inline_keyboard": inlineRows}
			sendMessageWithReplyMarkup(chatID, payText, reply)
			// notify admins about new user registration
			adminMsg := fmt.Sprintf(t("admin_new_user"), user.Email, user.UUID)
			sendToAdmins(adminMsg)
			clearConversation(chatID)
			return true
		}
	}
	if c.Action == "login" {
		if c.Step == 0 {
			c.Email = strings.TrimSpace(text)
			c.Step = 1
			setConversation(chatID, c)
			sendMessage(chatID, t("enter_password"))
			return true
		}
		if c.Step == 1 {
			password := strings.TrimSpace(text)
			_, _, err := botService.Auth.AuthenticateUser(c.Email, password)
			if err != nil {
				sendMessage(chatID, fmt.Sprintf(t("login_error"), err))
				clearConversation(chatID)
				return true
			}
			user, err := botService.UserRepo.GetUserByEmail(c.Email)
			if err != nil {
				sendMessage(chatID, fmt.Sprintf(t("user_not_found_after_auth"), err))
				clearConversation(chatID)
				return true
			}
			if err := botService.UserRepo.UpdateUserTelegramID(int(user.ID), fromID); err != nil {
				sendMessage(chatID, fmt.Sprintf(t("link_telegram_error"), err))
				clearConversation(chatID)
				return true
			}
			sendMessage(chatID, t("login_success"))
			clearConversation(chatID)
			return true
		}
	}
	return false
}

// Handlers (register/login/updatepassword/start/tariffs/pay)
func handleRegister(chatID int64, args []string) {
	if len(args) < 2 {
		sendMessage(chatID, t("usage_register"))
		return
	}
	email, password := args[0], args[1]
	uuidStr := uuid.NewString()
	user, err := botService.Auth.Register(email, password, uuidStr, 0)
	if err != nil {
		sendMessage(chatID, fmt.Sprintf("Error registering user: %s", err))
		return
	}
	sendMessage(chatID, fmt.Sprintf("User %s registered successfully!", user.Email))
}

func handleLogin(chatID int64, fromID int64, args []string) {
	if len(args) < 2 {
		sendMessage(chatID, t("usage_login"))
		return
	}
	identifier, password := args[0], args[1]
	_, _, err := botService.Auth.AuthenticateUser(identifier, password)
	if err != nil {
		sendMessage(chatID, fmt.Sprintf("Error logging in: %s", err))
		return
	}
	user, err := botService.UserRepo.GetUserByEmail(identifier)
	if err != nil {
		sendMessage(chatID, fmt.Sprintf("Error finding user: %s", err))
		return
	}
	if err := botService.UserRepo.UpdateUserTelegramID(int(user.ID), fromID); err != nil {
		sendMessage(chatID, fmt.Sprintf("Error linking Telegram ID: %s", err))
		return
	}
	sendMessage(chatID, fmt.Sprintf("User %s logged in and linked successfully!", user.Email))
}

func handleUpdatePassword(chatID int64, fromID int64, args []string) {
	// Two modes supported:
	// 1) If user is linked by telegram and provides: /updatepassword <old_password> <new_password>
	// 2) If not linked (or admin flow): /updatepassword <email> <old_password> <new_password>
	if len(args) == 2 {
		// try to resolve user by telegram id
		user, err := botService.UserRepo.GetUserByTelegramID(fromID)
		if err != nil {
			sendMessage(chatID, "Usage: /updatepassword <email> <old_password> <new_password> — or link your account via /start <token> first")
			return
		}
		oldPass := args[0]
		newPass := args[1]
		if err := botService.UserRepo.UpdatePasswordIfMatchesUserID(int(user.ID), oldPass, newPass); err != nil {
			sendMessage(chatID, fmt.Sprintf("Error updating password: %s", err))
			return
		}
		sendMessage(chatID, "Password updated successfully!")
		return
	}

	if len(args) == 3 {
		email := args[0]
		oldPass := args[1]
		newPass := args[2]
		if err := botService.UserRepo.UpdatePasswordIfMatchesEmail(email, oldPass, newPass); err != nil {
			sendMessage(chatID, fmt.Sprintf("Error updating password: %s", err))
			return
		}
		sendMessage(chatID, "Password updated successfully!")
		return
	}

	sendMessage(chatID, "Usage: /updatepassword <old_password> <new_password> (when account linked with bot)\nor /updatepassword <email> <old_password> <new_password>")
}

func handleStart(chatID int64, fromID int64, args []string) {
	if len(args) < 1 {
		sendMessage(chatID, "Usage: /start <token>")
		return
	}
	token := args[0]
	userID, err := botService.Auth.ValidateBotToken(token)
	if err != nil {
		// token might be a plain UUID generated by the client app. Try to link by UUID.
		user, uerr := botService.UserRepo.GetUserByUUID(token)
		if uerr != nil {
			sendMessage(chatID, "Invalid or expired token. Please request the bot link from the app again.")
			return
		}
		// Link user by UUID
		if err := botService.UserRepo.UpdateUserTelegramID(int(user.ID), fromID); err != nil {
			sendMessage(chatID, fmt.Sprintf("Failed to link account: %v", err))
			return
		}
		// Respond as linked and offer subscription buttons
		welcome := t("welcome_linked")
		keyboard := [][]string{{"/tariffs", "/pay"}, {"/me", "/offer"}, {"/help"}}
		sendMessageWithKeyboard(chatID, welcome, keyboard)
		// Inline buttons for subscriptions (1 month, 3 months)
		payText := t("choose_subscription")
		inlineRows := [][]map[string]interface{}{
			{{"text": t("subscription_1"), "callback_data": "robokassa_buy:1"}},
			{{"text": t("subscription_3"), "callback_data": "robokassa_buy:3"}},
		}
		reply := map[string]interface{}{"inline_keyboard": inlineRows}
		sendMessageWithReplyMarkup(chatID, payText, reply)
		sendMessage(chatID, t("use_tariffs"))
		return
	}
	if err := botService.UserRepo.UpdateUserTelegramID(userID, fromID); err != nil {
		sendMessage(chatID, fmt.Sprintf("Failed to link account: %v", err))
		return
	}
	// friendly greeting and quick-action keyboard; offer subscriptions inline
	welcome := t("welcome_linked")
	keyboard := [][]string{{"/tariffs", "/pay"}, {"/me", "/offer"}, {"/help"}}
	sendMessageWithKeyboard(chatID, welcome, keyboard)
	// Inline subscription buttons
	payText := t("choose_subscription")
	inlineRows := [][]map[string]interface{}{
		{{"text": t("subscription_1"), "callback_data": "robokassa_buy:1"}},
		{{"text": t("subscription_3"), "callback_data": "robokassa_buy:3"}},
	}
	reply := map[string]interface{}{"inline_keyboard": inlineRows}
	sendMessageWithReplyMarkup(chatID, payText, reply)
	sendMessage(chatID, t("use_tariffs"))
}

func handleTariffs(chatID int64) {
	tariffs, err := botService.TariffRepo.GetAll()
	if err != nil {
		sendMessage(chatID, t("tariffs_fetch_error"))
		return
	}
	if len(tariffs) == 0 {
		sendMessage(chatID, t("tariffs_empty"))
		return
	}
	var b strings.Builder
	b.WriteString(t("tariffs_list_header") + "\n")
	for _, t := range tariffs {
		b.WriteString(fmt.Sprintf("%d) %s — %.0f RUB — %d MB\n", t.ID, t.Name, t.Price, t.TrafficLimit/1024/1024))
	}
	b.WriteString(t("tariffs_pay_instruction"))
	// Additionally provide an inline keyboard with Buy buttons
	// build inline keyboard structure
	var inlineRows [][]map[string]interface{}
	for _, t := range tariffs {
		btn := map[string]interface{}{"text": fmt.Sprintf("Купить %s — %.0f₽", t.Name, t.Price), "callback_data": fmt.Sprintf("buy:%d", t.ID)}
		inlineRows = append(inlineRows, []map[string]interface{}{btn})
	}
	reply := map[string]interface{}{"inline_keyboard": inlineRows}
	sendMessageWithReplyMarkup(chatID, b.String(), reply)
}

// simple localization map and helper
var messages = map[string]map[string]string{
	"ru": {
		"enter_email":                           "Введите email",
		"enter_password":                        "Введите пароль",
		"usage_register":                        "Использование: /register <email> <password>",
		"usage_login":                           "Использование: /login <email> <password>",
		"usage_pay":                             "Использование: /pay <tariff_id>",
		"tariffs_fetch_error":                   "Не удалось получить список тарифов.",
		"tariffs_empty":                         "Тарифы не настроены.",
		"tariffs_list_header":                   "Доступные тарифы:",
		"tariffs_pay_instruction":               "Оплатите тариф командой: /pay <tariff_id>",
		"registration_success":                  "Регистрация успешна и аккаунт привязан к Telegram.",
		"registration_error":                    "Ошибка регистрации: %v",
		"login_success":                         "Вход успешен и аккаунт привязан к Telegram.",
		"login_error":                           "Ошибка входа: %v",
		"invalid_tariff_id":                     "Неверный ID тарифа",
		"account_not_linked":                    "Аккаунт не привязан. Пожалуйста, выполните /start <token> в боте сначала.",
		"tariff_not_found":                      "Тариф не найден",
		"public_offer":                          "Публичная оферта",
		"pay_prompt":                            "Оплатить можно через кнопки ниже или командой /pay <tariff_id>",
		"help_text":                             "Доступные команды: /start <token>, /tariffs, /pay <tariff_id>, /register, /login, /me, /offer",
		"user_not_found_after_auth":             "Пользователь не найден после успешной аутентификации: %v",
		"link_telegram_error":                   "Не удалось привязать Telegram: %v",
		"welcome_linked":                        "Аккаунт успешно привязан. Добро пожаловать! Ниже — быстрые команды и варианты подписки.",
		"choose_subscription":                   "Выберите подписку",
		"subscription_1":                        "Подписка 1 месяц — 200₽",
		"subscription_3":                        "Подписка 3 месяца — 500₽",
		"use_tariffs":                           "Или используйте /tariffs для списка тарифов",
		"offer_unavailable":                     "Публичная оферта недоступна",
		"public_offer_link":                     "Публичная оферта: %s",
		"payment_failed_fmt":                    "Не удалось создать платёж: %v",
		"payment_link":                          "Ссылка на оплату: %s",
		"payment_provider_id":                   "Идентификатор платёжной системы: %s",
		"receipt_amount":                        "Чек: сумма %d ₽.",
		"receipt_amount_with_inv":               "Чек: сумма %d ₽. InvId: %s",
		"cb_invalid_command":                    "Неверная команда",
		"cb_invalid_tariff_id":                  "Неверный id тарифа",
		"cb_account_not_linked":                 "Аккаунт не привязан. Откройте бота через приложение и привяжите аккаунт.",
		"cb_payment_failed":                     "Не удалось создать платёж",
		"cb_payment_created_ack":                "Платёж создан, проверьте ссылку в чате",
		"cb_invalid_period":                     "Неверный период подписки",
		"cb_server_not_configured_for_payments": "Сервер не настроен для оплат",
		"expiry_unknown":                        "неизвестно",
		"no_subscription":                       "нет подписки",
		"unknown_command":                       "Неизвестная команда. Введите /help для списка команд.",
		"payment_created":                       "Платёж создан. Перейдите по ссылке для оплаты: %s",
		"admin_new_user":                        "Новый пользователь зарегистрирован: %s (uuid=%s)",
	},
}

func t(key string) string {
	lang := "ru"
	if botService != nil && botService.Config != nil && botService.Config.DefaultLang != "" {
		lang = botService.Config.DefaultLang
	}
	if m, ok := messages[lang]; ok {
		if v, ok2 := m[key]; ok2 {
			return v
		}
	}
	return key
}

func handlePay(chatID int64, fromID int64, args []string) {
	if len(args) < 1 {
		sendMessage(chatID, t("usage_pay"))
		return
	}
	tariffID, err := strconv.Atoi(args[0])
	if err != nil {
		sendMessage(chatID, t("invalid_tariff_id"))
		return
	}
	user, err := botService.UserRepo.GetUserByTelegramID(fromID)
	if err != nil {
		sendMessage(chatID, t("account_not_linked"))
		return
	}
	tariff, err := botService.TariffRepo.FindByID(tariffID)
	if err != nil {
		sendMessage(chatID, t("tariff_not_found"))
		return
	}
	amountRub := int(tariff.Price)
	description := fmt.Sprintf("%s — подписка %s", user.Email, tariff.Name)
	confirmURL, providerID, err := botService.Payment.CreateYooKassaPayment(int(user.ID), 1, botService.Config.YooKassaReturnURL, botService.Config.YooKassaShopID, botService.Config.YooKassaSecret, amountRub, description)
	if err != nil {
		sendMessage(chatID, fmt.Sprintf(t("payment_failed_fmt"), err))
		return
	}
	sendMessage(chatID, fmt.Sprintf(t("payment_created"), confirmURL))
	// Send receipt to the user chat (provider id + amount)
	sendMessage(chatID, fmt.Sprintf(t("payment_link"), confirmURL))
	// If providerID is returned include it in the receipt
	if providerID != "" {
		sendMessage(chatID, fmt.Sprintf(t("payment_provider_id"), providerID))
	}
	// Notify administrators in configured admin chats with details (email, amount, provider, link)
	adminMsg := fmt.Sprintf("Платёж создан пользователем %s на %d ₽. Ссылка: %s", user.Email, amountRub, confirmURL)
	sendToAdmins(adminMsg)
}

// handleCallback processes inline callback queries (e.g., buy:<tariff_id>)
func handleCallback(cb *tgCallbackQuery) {
	if cb == nil {
		return
	}
	data := cb.Data
	if strings.HasPrefix(data, "buy:") {
		parts := strings.SplitN(data, ":", 2)
		if len(parts) != 2 {
			answerCallback(cb.ID, t("cb_invalid_command"))
			return
		}
		tid, err := strconv.Atoi(parts[1])
		if err != nil {
			answerCallback(cb.ID, t("cb_invalid_tariff_id"))
			return
		}
		// Try to find user by telegram id
		user, err := botService.UserRepo.GetUserByTelegramID(cb.From.ID)
		if err != nil {
			answerCallback(cb.ID, t("cb_account_not_linked"))
			return
		}
		tariff, err := botService.TariffRepo.FindByID(tid)
		if err != nil {
			answerCallback(cb.ID, "Тариф не найден")
			return
		}
		amountRub := int(tariff.Price)
		description := fmt.Sprintf("%s — подписка %s", user.Email, tariff.Name)
		confirmURL, providerID, err := botService.Payment.CreateYooKassaPayment(int(user.ID), 1, botService.Config.YooKassaReturnURL, botService.Config.YooKassaShopID, botService.Config.YooKassaSecret, amountRub, description)
		if err != nil {
			answerCallback(cb.ID, t("cb_payment_failed"))
			return
		}
		// Acknowledge the button press and send link
		answerCallback(cb.ID, t("cb_payment_created_ack"))
		sendMessage(cb.Message.Chat.ID, fmt.Sprintf(t("payment_created"), confirmURL))
		// Send receipt to the user chat
		sendMessage(cb.Message.Chat.ID, fmt.Sprintf(t("receipt_amount"), amountRub))
		if providerID != "" {
			sendMessage(cb.Message.Chat.ID, fmt.Sprintf(t("payment_provider_id"), providerID))
		}
		// Notify administrators in configured admin chats
		adminMsg := fmt.Sprintf("Платёж создан пользователем %s на %d ₽. ProviderID: %s. Ссылка: %s", user.Email, amountRub, providerID, confirmURL)
		sendToAdmins(adminMsg)
	} else if strings.HasPrefix(data, "robokassa_buy:") {
		parts := strings.SplitN(data, ":", 2)
		if len(parts) != 2 {
			answerCallback(cb.ID, t("cb_invalid_command"))
			return
		}
		months, err := strconv.Atoi(parts[1])
		if err != nil || (months != 1 && months != 3) {
			answerCallback(cb.ID, t("cb_invalid_period"))
			return
		}
		// Find user by telegram id
		user, err := botService.UserRepo.GetUserByTelegramID(cb.From.ID)
		if err != nil {
			answerCallback(cb.ID, t("cb_account_not_linked"))
			return
		}
		// Determine amounts (as requested): 1 month = 200, 3 months = 500
		amount := 200
		if months == 3 {
			amount = 500
		}
		description := fmt.Sprintf("%s — подписка %d мес.", user.Email, months)
		// Create Robokassa payment (uses Robokassa config from server)
		if botService.Config == nil {
			answerCallback(cb.ID, t("cb_server_not_configured_for_payments"))
			return
		}
		confirmURL, invId, successURL, failURL, err := botService.Payment.CreateRobokassaPayment(int(user.ID), months, amount, description, botService.Config.RobokassaLogin, botService.Config.RobokassaPassword1, botService.Config.FrontendURL)
		if err != nil {
			answerCallback(cb.ID, t("cb_payment_failed"))
			return
		}
		answerCallback(cb.ID, t("cb_payment_created_ack"))
		sendMessage(cb.Message.Chat.ID, fmt.Sprintf(t("payment_created"), confirmURL))
		sendMessage(cb.Message.Chat.ID, fmt.Sprintf(t("receipt_amount_with_inv"), amount, invId))
		// Notify admins
		adminMsg := fmt.Sprintf("Платёж создан пользователем %s на %d ₽. InvId: %s. Ссылка: %s. Return success: %s, fail: %s", user.Email, amount, invId, confirmURL, successURL, failURL)
		sendToAdmins(adminMsg)
	} else {
		answerCallback(cb.ID, "Неизвестная команда")
	}
}

func handleMe(chatID int64, fromID int64) {
	user, err := botService.UserRepo.GetUserByTelegramID(fromID)
	if err != nil {
		sendMessage(chatID, "Аккаунт не привязан. Пожалуйста, выполните /start <token> в боте сначала.")
		return
	}
	// Try to get tariff expiry info
	expiry, err := botService.Payment.GetTariffExpiry(int(user.ID))
	var expiryStr string
	if err != nil {
		expiryStr = t("expiry_unknown")
	} else if expiry.IsZero() {
		expiryStr = t("no_subscription")
	} else {
		expiryStr = expiry.Format("2006-01-02 15:04:05")
	}
	// Build info
	info := fmt.Sprintf("Информация о пользователе:\nEmail: %s\nПодписка до: %s\nИспользовано трафика: %d байт", user.Email, expiryStr, user.UsedTraffic)
	sendMessage(chatID, info)
}

func handleOffer(chatID int64) {
	if botService == nil || botService.Config == nil {
		sendMessage(chatID, "Публичная оферта недоступна")
		return
	}
	offer := botService.Config.PublicOfferURL
	if offer == "" {
		// fallback to frontend URL + /offer
		if botService.Config.FrontendURL != "" {
			offer = strings.TrimRight(botService.Config.FrontendURL, "/") + "/offer"
		}
	}
	if offer == "" {
		sendMessage(chatID, "Публичная оферта не настроена на сервере")
		return
	}
	sendMessage(chatID, fmt.Sprintf("Публичная оферта: %s", offer))
}

func StartBot() error {
	if botService == nil {
		return fmt.Errorf("bot is not initialized")
	}
	offset := 0
	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=60&offset=%d", botService.Token, offset)
		resp, err := http.Get(url)
		if err != nil {
			botService.Logger.Printf("getUpdates error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var ur tgUpdateResp
		if err := json.Unmarshal(body, &ur); err != nil {
			botService.Logger.Printf("invalid updates response: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		for _, u := range ur.Result {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			// handle callback_query first
			if u.CallbackQuery != nil {
				go handleCallback(u.CallbackQuery)
				continue
			}
			if u.Message == nil || u.Message.Text == "" {
				continue
			}
			text := u.Message.Text
			// If there is an active conversation state, handle it first
			if handled := handleStateMessage(u.Message.Chat.ID, u.Message.From.ID, text); handled {
				continue
			}

			// Map some friendly button labels to commands
			lowered := strings.TrimSpace(text)
			switch lowered {
			case "Зарегистрироваться", "зарегистрироваться":
				// start interactive register flow
				setConversation(u.Message.Chat.ID, &conversation{Action: "register", Step: 0})
				sendMessage(u.Message.Chat.ID, t("enter_email"))
				continue
			case "Войти", "войти":
				setConversation(u.Message.Chat.ID, &conversation{Action: "login", Step: 0})
				sendMessage(u.Message.Chat.ID, t("enter_email"))
				continue
			case "Профиль", "профиль":
				handleMe(u.Message.Chat.ID, u.Message.From.ID)
				continue
			case "Правила", "правила", "Оферта", "оферта":
				// send offer file if configured; fallback to URL
				if botService != nil && botService.Config != nil && botService.Config.PublicOfferURL != "" {
					sendDocumentURL(u.Message.Chat.ID, botService.Config.PublicOfferURL, t("public_offer"))
				} else {
					handleOffer(u.Message.Chat.ID)
				}
				continue
			case "Оплатить", "оплатить":
				sendMessageWithKeyboard(u.Message.Chat.ID, t("pay_prompt"), [][]string{{"/tariffs", "/pay"}})
				continue
			}

			cmd, args := parseCommand(text)
			switch cmd {
			case "/start":
				handleStart(u.Message.Chat.ID, u.Message.From.ID, args)
			case "/help":
				helpText := t("help_text")
				sendMessageWithKeyboard(u.Message.Chat.ID, helpText, [][]string{{"/tariffs", "/pay"}, {"/me", "/offer"}, {"/register", "/login"}})
			case "/offer":
				handleOffer(u.Message.Chat.ID)
			case "/me":
				handleMe(u.Message.Chat.ID, u.Message.From.ID)
			case "/register":
				// start interactive flow
				setConversation(u.Message.Chat.ID, &conversation{Action: "register", Step: 0})
				sendMessage(u.Message.Chat.ID, t("enter_email"))
			case "/login":
				setConversation(u.Message.Chat.ID, &conversation{Action: "login", Step: 0})
				sendMessage(u.Message.Chat.ID, t("enter_email"))
			case "/updatepassword":
				handleUpdatePassword(u.Message.Chat.ID, u.Message.From.ID, args)
			case "/tariffs":
				handleTariffs(u.Message.Chat.ID)
			case "/pay":
				handlePay(u.Message.Chat.ID, u.Message.From.ID, args)
			default:
				sendMessage(u.Message.Chat.ID, t("unknown_command"))
			}
		}
	}
}
