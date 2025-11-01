package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/yourusername/vpn-backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) Delete(userID int) error {
	result := r.DB.Unscoped().Delete(&models.User{}, userID)
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
	result := r.DB.Preload("Tariff").First(&user, userID) // Добавьте Preload
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

func (r *UserRepository) ResetAllTraffic() error {
	result := r.DB.Model(&models.User{}).Update("used_traffic", 0)
	return result.Error
}

func (r *UserRepository) GetPaymentsByUserID(userID int) ([]models.Payment, error) {
	var payments []models.Payment
	result := r.DB.Where("user_id = ?", userID).Find(&payments)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get payments: %w", result.Error)
	}
	return payments, nil
}

func (r *UserRepository) GetPaymentByID(userID int, paymentID string) (*models.Payment, error) {
	var payment models.Payment
	result := r.DB.Where("id = ? AND user_id = ?", paymentID, userID).First(&payment)
	if result.Error != nil {
		return nil, fmt.Errorf("payment not found: %w", result.Error)
	}
	return &payment, nil
}

func (r *UserRepository) CreatePayment(payment *models.Payment) error {
	result := r.DB.Create(payment)
	if result.Error != nil {
		return fmt.Errorf("failed to create payment: %w", result.Error)
	}
	return nil
}

func (r *UserRepository) UpdatePaymentStatus(userID int, paymentID string, status string) error {
	result := r.DB.Model(&models.Payment{}).Where("id = ? AND user_id = ?", paymentID, userID).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update payment status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("payment not found")
	}
	return nil
}

func (r *UserRepository) UpdatePaymentProviderID(userID int, paymentID string, providerID string) error {
	result := r.DB.Model(&models.Payment{}).Where("user_id = ? AND id = ?", userID, paymentID).Update("provider_id", providerID)
	if result.Error != nil {
		return fmt.Errorf("failed to update payment provider id: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("payment not found")
	}
	return nil
}

func (r *UserRepository) SetPasswordResetToken(email, token string, expiresAt time.Time) error {
	return r.DB.Model(&models.User{}).Where("email = ?", email).
		Updates(map[string]interface{}{
			"password_reset_token":      token,
			"password_reset_expires_at": expiresAt,
		}).Error
}

func (r *UserRepository) SetRefreshToken(userID int, tokenHash string, expiresAt time.Time) error {
	return r.DB.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"refresh_token_hash":       tokenHash,
			"refresh_token_expires_at": expiresAt,
		}).Error
}

func (r *UserRepository) ClearRefreshToken(userID int) error {
	return r.DB.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"refresh_token_hash":       "",
			"refresh_token_expires_at": time.Time{},
		}).Error
}

func (r *UserRepository) FindByRefreshToken(token string) (*models.User, error) {
	var user models.User
	// We store only the SHA256 hex of the refresh token. Compute and lookup.
	h := sha256.Sum256([]byte(token))
	hashed := hex.EncodeToString(h[:])
	if err := r.DB.Where("refresh_token_hash = ? AND refresh_token_expires_at > ?", hashed, time.Now()).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByPasswordResetToken(token string) (*models.User, error) {
	var user models.User
	// First try exact match (to allow a short migration window for existing plain tokens)
	if err := r.DB.Where("password_reset_token = ? AND password_reset_expires_at > ?", token, time.Now()).First(&user).Error; err == nil {
		return &user, nil
	}

	// Otherwise, assume token was sent plain and we store SHA256 hex in DB — try hashed lookup
	h := sha256.Sum256([]byte(token))
	hashed := hex.EncodeToString(h[:])
	if err := r.DB.Where("password_reset_token = ? AND password_reset_expires_at > ?", hashed, time.Now()).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdatePasswordResetRequestedAt(email string, t time.Time) error {
	return r.DB.Model(&models.User{}).Where("email = ?", email).Update("password_reset_requested_at", t).Error
}

func (r *UserRepository) CountPasswordResetRequests(email string, since time.Time) (int64, error) {
	var count int64
	r.DB.Model(&models.User{}).Where("email = ? AND password_reset_requested_at > ?", email, since).Count(&count)
	return count, nil
}
