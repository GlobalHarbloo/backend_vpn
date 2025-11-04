package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/yourusername/vpn-backend/internal/models"
	"github.com/yourusername/vpn-backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	UserRepo  *repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		UserRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (a *AuthService) Register(email, password, uuid string, tariffID int) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:       email,
		Password:    string(hashedPassword),
		UUID:        uuid,
		TariffID:    tariffID,
		IsBanned:    false,
		UsedTraffic: 0,
		TrialEndsAt: time.Now().Add(72 * time.Hour),
	}

	if err := a.UserRepo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// AuthenticateUser authenticates credentials and returns access and refresh tokens.
func (a *AuthService) AuthenticateUser(email, password string) (string, string, error) {
	user, err := a.UserRepo.GetUserByEmail(email)
	if err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}

	// Create both access and refresh tokens
	access, refresh, err := a.CreateTokens(int(user.ID))
	if err != nil {
		return "", "", fmt.Errorf("failed to create tokens: %w", err)
	}
	return access, refresh, nil
}

func (a *AuthService) GenerateJWT(userID int) (string, error) {
	// Increase JWT lifetime to 7 days to avoid frequent forced re-login from mobile clients.
	// Consider implementing refresh tokens for a more secure long-lived session model.
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &jwt.MapClaims{
		"user_id": userID,
		"exp":     expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateBotToken creates a short-lived JWT intended for Telegram bot deep links.
// The token contains the user_id and expires after the provided ttl.
func (a *AuthService) GenerateBotToken(userID int, ttl time.Duration) (string, error) {
	expirationTime := time.Now().Add(ttl)
	claims := &jwt.MapClaims{
		"user_id": userID,
		"exp":     expirationTime.Unix(),
		"type":    "bot",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign bot token: %w", err)
	}
	return tokenString, nil
}

// CreateTokens generates an access JWT and a refresh token, stores the hashed refresh token in DB.
func (a *AuthService) CreateTokens(userID int) (accessToken string, refreshToken string, err error) {
	accessToken, err = a.GenerateJWT(userID)
	if err != nil {
		return "", "", err
	}

	// Generate secure random refresh token (64 hex chars -> 32 bytes)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshToken = hex.EncodeToString(b)

	// Hash and store
	h := sha256.Sum256([]byte(refreshToken))
	hashed := hex.EncodeToString(h[:])
	expiresAt := time.Now().AddDate(0, 1, 0) // refresh token valid 1 month
	if err := a.UserRepo.SetRefreshToken(userID, hashed, expiresAt); err != nil {
		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func ParseJWT(tokenString string, jwtSecret string) (int, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid user ID in token")
	}

	userID := int(userIDFloat)
	return userID, nil
}

func (a *AuthService) AuthenticateByTelegramID(telegramID int64) (string, string, error) {
	user, err := a.UserRepo.GetUserByTelegramID(telegramID)
	if err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}
	access, refresh, err := a.CreateTokens(int(user.ID))
	if err != nil {
		return "", "", fmt.Errorf("failed to create tokens: %w", err)
	}
	return access, refresh, nil
}

// ValidateBotToken parses a bot JWT token and ensures it's a bot-type token.
func (a *AuthService) ValidateBotToken(tokenString string) (int, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.jwtSecret), nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}
	// ensure token type is bot
	t, ok := claims["type"].(string)
	if !ok || t != "bot" {
		return 0, fmt.Errorf("not a bot token")
	}
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid user id in token")
	}
	return int(userIDFloat), nil
}
