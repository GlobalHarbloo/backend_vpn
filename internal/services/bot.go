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

// Handlers (register/login/updatepassword/start/tariffs/pay)
func handleRegister(chatID int64, args []string) {
	if len(args) < 2 {
		sendMessage(chatID, "Usage: /register <email> <password>")
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
		sendMessage(chatID, "Usage: /login <email> <password>")
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
		sendMessage(chatID, "Invalid or expired token. Please request the bot link from the app again.")
		return
	}
	if err := botService.UserRepo.UpdateUserTelegramID(userID, fromID); err != nil {
		sendMessage(chatID, fmt.Sprintf("Failed to link account: %v", err))
		return
	}
	// friendly greeting and quick-action keyboard
	welcome := "Аккаунт успешно привязан. Добро пожаловать! Ниже — быстрые команды."
	keyboard := [][]string{{"/tariffs", "/pay"}, {"/me", "/offer"}, {"/help"}}
	sendMessageWithKeyboard(chatID, welcome, keyboard)
	sendMessage(chatID, "Чтобы оплатить конкретный тариф, выполните: /pay <tariff_id>. Просмотреть тарифы: /tariffs")
}

func handleTariffs(chatID int64) {
	tariffs, err := botService.TariffRepo.GetAll()
	if err != nil {
		sendMessage(chatID, "Не удалось получить список тарифов.")
		return
	}
	if len(tariffs) == 0 {
		sendMessage(chatID, "Тарифы не настроены.")
		return
	}
	var b strings.Builder
	b.WriteString("Доступные тарифы:\n")
	for _, t := range tariffs {
		b.WriteString(fmt.Sprintf("%d) %s — %.0f RUB — %d MB\n", t.ID, t.Name, t.Price, t.TrafficLimit/1024/1024))
	}
	b.WriteString("Оплатите тариф командой: /pay <tariff_id>")
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

func handlePay(chatID int64, fromID int64, args []string) {
	if len(args) < 1 {
		sendMessage(chatID, "Usage: /pay <tariff_id>")
		return
	}
	tariffID, err := strconv.Atoi(args[0])
	if err != nil {
		sendMessage(chatID, "Invalid tariff id")
		return
	}
	user, err := botService.UserRepo.GetUserByTelegramID(fromID)
	if err != nil {
		sendMessage(chatID, "Аккаунт не привязан. Пожалуйста, выполните /start <token> в боте сначала.")
		return
	}
	tariff, err := botService.TariffRepo.FindByID(tariffID)
	if err != nil {
		sendMessage(chatID, "Тариф не найден")
		return
	}
	amountRub := int(tariff.Price)
	description := fmt.Sprintf("%s — подписка %s", user.Email, tariff.Name)
	confirmURL, providerID, err := botService.Payment.CreateYooKassaPayment(int(user.ID), 1, botService.Config.YooKassaReturnURL, botService.Config.YooKassaShopID, botService.Config.YooKassaSecret, amountRub, description)
	if err != nil {
		sendMessage(chatID, fmt.Sprintf("Не удалось создать платёж: %v", err))
		return
	}
	sendMessage(chatID, fmt.Sprintf("Платёж создан. Перейдите по ссылке для оплаты: %s", confirmURL))
	// Send receipt to the user chat (provider id + amount)
	sendMessage(chatID, fmt.Sprintf("Ссылка на оплату: %s", confirmURL))
	// If providerID is returned include it in the receipt
	if providerID != "" {
		sendMessage(chatID, fmt.Sprintf("Идентификатор платёжной системы: %s", providerID))
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
			answerCallback(cb.ID, "Неверная команда")
			return
		}
		tid, err := strconv.Atoi(parts[1])
		if err != nil {
			answerCallback(cb.ID, "Неверный id тарифа")
			return
		}
		// Try to find user by telegram id
		user, err := botService.UserRepo.GetUserByTelegramID(cb.From.ID)
		if err != nil {
			answerCallback(cb.ID, "Аккаунт не привязан. Откройте бота через приложение и привяжите аккаунт.")
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
			answerCallback(cb.ID, "Не удалось создать платёж")
			return
		}
		// Acknowledge the button press and send link
		answerCallback(cb.ID, "Платёж создан, проверьте ссылку в чате")
		sendMessage(cb.Message.Chat.ID, fmt.Sprintf("Платёж создан. Перейдите по ссылке для оплаты: %s", confirmURL))
		// Send receipt to the user chat
		sendMessage(cb.Message.Chat.ID, fmt.Sprintf("Чек: сумма %d ₽.", amountRub))
		if providerID != "" {
			sendMessage(cb.Message.Chat.ID, fmt.Sprintf("Идентификатор платёжной системы: %s", providerID))
		}
		// Notify administrators in configured admin chats
		adminMsg := fmt.Sprintf("Платёж создан пользователем %s на %d ₽. ProviderID: %s. Ссылка: %s", user.Email, amountRub, providerID, confirmURL)
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
		expiryStr = "неизвестно"
	} else if expiry.IsZero() {
		expiryStr = "нет подписки"
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
			cmd, args := parseCommand(text)
			switch cmd {
			case "/start":
				handleStart(u.Message.Chat.ID, u.Message.From.ID, args)
			case "/help":
				helpText := "Доступные команды: /start <token>, /tariffs, /pay <tariff_id>, /register <email> <password>, /login <email> <password>, /updatepassword, /me, /offer"
				sendMessageWithKeyboard(u.Message.Chat.ID, helpText, [][]string{{"/tariffs", "/pay"}, {"/me", "/offer"}, {"/register", "/login"}})
			case "/offer":
				handleOffer(u.Message.Chat.ID)
			case "/me":
				handleMe(u.Message.Chat.ID, u.Message.From.ID)
			case "/register":
				handleRegister(u.Message.Chat.ID, args)
			case "/login":
				handleLogin(u.Message.Chat.ID, u.Message.From.ID, args)
			case "/updatepassword":
				handleUpdatePassword(u.Message.Chat.ID, u.Message.From.ID, args)
			case "/tariffs":
				handleTariffs(u.Message.Chat.ID)
			case "/pay":
				handlePay(u.Message.Chat.ID, u.Message.From.ID, args)
			default:
				sendMessage(u.Message.Chat.ID, "Unknown command. Type /help for available commands.")
			}
		}
	}
}
