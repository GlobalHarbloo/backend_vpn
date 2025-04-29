package repository

import (
	"fmt"
	"time"
	"vpn-backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) Delete(userID int) error {
	result := r.DB.Delete(&models.User{}, userID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) CreateUser(user *models.User) error {
	result := r.DB.Create(user)
	if result.Error != nil {
		return fmt.Errorf("failed to create user: %w", result.Error)
	}
	return nil
}

func (r *UserRepository) FindByID(userID int) (*models.User, error) {
	var user models.User
	result := r.DB.First(&user, userID)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	result := r.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

func (r *UserRepository) GetUserByUUID(uuid string) (*models.User, error) {
	var user models.User
	result := r.DB.Where("uuid = ?", uuid).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

func (r *UserRepository) UpdateUserTariff(userID int, tariffID int) error {
	result := r.DB.Model(&models.User{}).Where("id = ?", userID).Update("tariff_id", tariffID)
	if result.Error != nil {
		return fmt.Errorf("failed to update user tariff: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	var users []models.User
	result := r.DB.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get all users: %w", result.Error)
	}
	return users, nil
}

func (r *UserRepository) BanUser(userID int, ban bool) error {
	result := r.DB.Model(&models.User{}).Where("id = ?", userID).Update("is_banned", ban)
	if result.Error != nil {
		return fmt.Errorf("failed to ban user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) GetUserByTelegramID(telegramID int64) (*models.User, error) {
	var user models.User
	result := r.DB.Where("telegram_id = ?", telegramID).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

func (r *UserRepository) UpdateUserTelegramID(userID int, telegramID int64) error {
	result := r.DB.Model(&models.User{}).Where("id = ?", userID).Update("telegram_id", telegramID)
	if result.Error != nil {
		return fmt.Errorf("failed to update user telegram_id: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) UpdateTariffExpiry(userID int, expiryDate time.Time) error {
	result := r.DB.Model(&models.User{}).Where("id = ?", userID).Update("tariff_expires_at", expiryDate)
	if result.Error != nil {
		return fmt.Errorf("failed to update tariff expiry: %w", result.Error)
	}
	return nil
}

func (r *UserRepository) GetTariffExpiry(userID int) (time.Time, error) {
	var user models.User
	result := r.DB.Select("tariff_expires_at").First(&user, userID)
	if result.Error != nil {
		return time.Time{}, fmt.Errorf("failed to get tariff expiry: %w", result.Error)
	}
	return user.TariffExpiresAt, nil
}

func (r *UserRepository) UpdateUsedTraffic(userID int, traffic int64) error {
	result := r.DB.Model(&models.User{}).Where("id = ?", userID).Update("used_traffic", traffic)
	if result.Error != nil {
		return fmt.Errorf("failed to update used traffic: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
