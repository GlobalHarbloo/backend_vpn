package services

import (
	"errors"
	"os"
	"time"
	"vpn-backend/internal/models"
	"vpn-backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET")) // Читаем из переменных окружения

type AuthService struct {
	Repo *repository.UserRepository
}

func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{Repo: repo}
}

// ✅ Регистрация по полям (используется в ручке)
func (a *AuthService) Register(email, password, uuid string, tariffID int) (*models.User, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Email:    email,
		Password: string(hashedPwd),
		UUID:     uuid,
		TariffID: tariffID,
	}
	err = a.Repo.CreateUser(user)
	return user, err
}

// ✅ Метод для регистрации из User модели
func (a *AuthService) RegisterUser(user *models.User) error {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPwd)
	return a.Repo.CreateUser(user)
}

// ✅ Метод логина
func (a *AuthService) AuthenticateUser(email, password string) (string, error) {
	user, err := a.Repo.GetUserByEmail(email)
	if err != nil || user == nil {
		return "", errors.New("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString(jwtKey)
	return tokenStr, err
}

// ✅ Парсинг JWT
func ParseJWT(tokenStr string) (int, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return 0, err
	}
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid token")
	}
	return int(userID), nil
}
