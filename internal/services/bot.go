package services

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"github.com/yourusername/vpn-backend/config"
	"github.com/yourusername/vpn-backend/internal/models"
	"github.com/yourusername/vpn-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// BotService holds bot and required dependencies.
type BotService struct {
	BotAPI     *tgbotapi.BotAPI
	Logger     *log.Logger
	UserRepo   *repository.UserRepository
	Auth       *AuthService
	Payment    *PaymentService
	TariffRepo *repository.TariffRepository
	Config     *config.Config
}

var botService *BotService

// InitBot initializes the Telegram bot and stores dependencies.
func InitBot(token string, userRepo *repository.UserRepository, auth *AuthService, payment *PaymentService, tariffRepo *repository.TariffRepository, cfg *config.Config) error {
	b, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	logger := log.New(log.Writer(), "bot: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Println("Bot successfully initialized.")

	botService = &BotService{
		BotAPI:     b,
		Logger:     logger,
		UserRepo:   userRepo,
		Auth:       auth,
		Payment:    payment,
		TariffRepo: tariffRepo,
		Config:     cfg,
	}
	return nil
}

// HandleCommands routes incoming updates to handlers.
func HandleCommands(update tgbotapi.Update) {
	if botService == nil || update.Message == nil || update.Message.Text == "" {
		return
	}

	command, args := parseCommand(update.Message.Text)

	switch command {
	case "/start":
		handleStart(update, args)
	case "/help":
		sendMessage(update.Message.Chat.ID, "Available commands: /start <token>, /tariffs, /pay <tariff_id>, /register <email> <password>, /login <email> <password>, /updatepassword <email> <new_password>")
	case "/register":
		handleRegister(update, args)
	case "/login":
		handleLogin(update, args)
	case "/updatepassword":
		handleUpdatePassword(update, args)
	case "/tariffs":
		handleTariffs(update)
	case "/pay":
		handlePay(update, args)
	default:
		sendMessage(update.Message.Chat.ID, "Unknown command. Type /help for available commands.")
	}
}

// parseCommand splits the message into command and args.
func parseCommand(input string) (string, []string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

// sendMessage sends a text message and logs failures.
func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := botService.BotAPI.Send(msg); err != nil {
		botService.Logger.Printf("Failed to send message: %v\n", err)
	}
}

// handleRegister creates a new user via AuthService.Register.
// Usage: /register <email> <password>
func handleRegister(update tgbotapi.Update, args []string) {
	if len(args) < 2 {
		sendMessage(update.Message.Chat.ID, "Usage: /register <email> <password>")
		return
	}

	email, password := args[0], args[1]
	// generate UUID for the user
	uuidStr := uuid.NewString()
	user, err := botService.Auth.Register(email, password, uuidStr, 0)
	if err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Error registering user: %s", err))
		return
	}

	sendMessage(update.Message.Chat.ID, fmt.Sprintf("User %s registered successfully!", user.Email))
}

// handleLogin authenticates user credentials and links Telegram ID.
// Usage: /login <email> <password>
func handleLogin(update tgbotapi.Update, args []string) {
	if len(args) < 2 {
		sendMessage(update.Message.Chat.ID, "Usage: /login <email> <password>")
		return
	}

	identifier, password := args[0], args[1]
	_, _, err := botService.Auth.AuthenticateUser(identifier, password)
	if err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Error logging in: %s", err))
		return
	}

	// find user and link telegram id
	user, err := botService.UserRepo.GetUserByEmail(identifier)
	if err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Error finding user: %s", err))
		return
	}

	if err := botService.UserRepo.UpdateUserTelegramID(int(user.ID), int64(update.Message.From.ID)); err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Error linking Telegram ID: %s", err))
		return
	}

	sendMessage(update.Message.Chat.ID, fmt.Sprintf("User %s logged in successfully!", user.Email))
}

// handleUpdatePassword updates a user's password by email.
// Usage: /updatepassword <email> <new_password>
func handleUpdatePassword(update tgbotapi.Update, args []string) {
	if len(args) < 2 {
		sendMessage(update.Message.Chat.ID, "Usage: /updatepassword <email> <new_password>")
		return
	}

	email, newPassword := args[0], args[1]
	// find user
	user, err := botService.UserRepo.GetUserByEmail(email)
	if err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Error finding user: %s", err))
		return
	}

	// hash password and update
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Error hashing password: %s", err))
		return
	}

	user.Password = string(hashed)
	if err := botService.UserRepo.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("password", user.Password).Error; err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Error updating password: %s", err))
		return
	}

	sendMessage(update.Message.Chat.ID, "Password updated successfully!")
}

// handleStart validates bot deep link token, links telegram id to user, and shows payment options
// Usage: /start <token>
func handleStart(update tgbotapi.Update, args []string) {
	if len(args) < 1 {
		sendMessage(update.Message.Chat.ID, "Usage: /start <token>")
		return
	}
	token := args[0]
	userID, err := botService.Auth.ValidateBotToken(token)
	if err != nil {
		sendMessage(update.Message.Chat.ID, "Invalid or expired token. Please request the bot link from the app again.")
		return
	}

	// Link telegram ID
	if err := botService.UserRepo.UpdateUserTelegramID(userID, int64(update.Message.From.ID)); err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Failed to link account: %v", err))
		return
	}

	sendMessage(update.Message.Chat.ID, "Аккаунт успешно привязан. Чтобы оплатить подписку, используйте команду: /tariffs — посмотреть тарифы, /pay <tariff_id> — оплатить.")
}

// handleTariffs lists available tariffs
func handleTariffs(update tgbotapi.Update) {
	tariffs, err := botService.TariffRepo.GetAll()
	if err != nil {
		sendMessage(update.Message.Chat.ID, "Не удалось получить список тарифов.")
		return
	}
	if len(tariffs) == 0 {
		sendMessage(update.Message.Chat.ID, "Тарифы не настроены.")
		return
	}
	var b strings.Builder
	b.WriteString("Доступные тарифы:\n")
	for _, t := range tariffs {
		b.WriteString(fmt.Sprintf("%d) %s — %.0f RUB — %d MB\n", t.ID, t.Name, t.Price, t.TrafficLimit/1024/1024))
	}
	b.WriteString("Оплатите тариф командой: /pay <tariff_id>")
	sendMessage(update.Message.Chat.ID, b.String())
}

// handlePay creates YooKassa payment and returns confirmation URL
// Usage: /pay <tariff_id>
func handlePay(update tgbotapi.Update, args []string) {
	if len(args) < 1 {
		sendMessage(update.Message.Chat.ID, "Usage: /pay <tariff_id>")
		return
	}
	tid := args[0]
	tariffID, err := strconv.Atoi(tid)
	if err != nil {
		sendMessage(update.Message.Chat.ID, "Invalid tariff id")
		return
	}
	// find user by telegram id
	user, err := botService.UserRepo.GetUserByTelegramID(int64(update.Message.From.ID))
	if err != nil {
		sendMessage(update.Message.Chat.ID, "Аккаунт не привязан. Пожалуйста, выполните /start <token> в боте сначала.")
		return
	}

	tariff, err := botService.TariffRepo.FindByID(tariffID)
	if err != nil {
		sendMessage(update.Message.Chat.ID, "Тариф не найден")
		return
	}

	amountRub := int(tariff.Price)
	description := fmt.Sprintf("%s — подписка %s", user.Email, tariff.Name)
	// Create YooKassa payment
	confirmURL, _, err := botService.Payment.CreateYooKassaPayment(int(user.ID), 1, botService.Config.YooKassaReturnURL, botService.Config.YooKassaShopID, botService.Config.YooKassaSecret, amountRub, description)
	if err != nil {
		sendMessage(update.Message.Chat.ID, fmt.Sprintf("Не удалось создать платёж: %v", err))
		return
	}
	// Send confirmation URL to user
	sendMessage(update.Message.Chat.ID, fmt.Sprintf("Платёж создан. Перейдите по ссылке для оплаты: %s", confirmURL))
}

// StartBot starts listening for updates and dispatches them to the handler.
func StartBot() error {
	if botService == nil {
		return fmt.Errorf("bot is not initialized")
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := botService.BotAPI.GetUpdatesChan(u)

	for update := range updates {
		HandleCommands(update)
	}
	return nil
}
