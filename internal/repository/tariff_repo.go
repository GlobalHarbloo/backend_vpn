package repository

import (
	"fmt"
	"vpn-backend/internal/models"

	"gorm.io/gorm"
)

type TariffRepository struct {
	DB *gorm.DB
}

func NewTariffRepository(db *gorm.DB) *TariffRepository {
	return &TariffRepository{DB: db}
}

func (r *TariffRepository) Create(tariff *models.Tariff) error {
	result := r.DB.Create(tariff)
	if result.Error != nil {
		return fmt.Errorf("failed to create tariff: %w", result.Error)
	}
	return nil
}

func (r *TariffRepository) FindByID(id int) (*models.Tariff, error) {
	var tariff models.Tariff
	result := r.DB.First(&tariff, id)
	if result.Error != nil {
		return nil, fmt.Errorf("tariff not found: %w", result.Error)
	}
	return &tariff, nil
}

func (r *TariffRepository) GetAll() ([]models.Tariff, error) {
	var tariffs []models.Tariff
	result := r.DB.Find(&tariffs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get tariffs: %w", result.Error)
	}
	return tariffs, nil
}

func (r *TariffRepository) Update(tariff *models.Tariff) error {
	result := r.DB.Save(tariff)
	if result.Error != nil {
		return fmt.Errorf("failed to update tariff: %w", result.Error)
	}
	return nil
}

func (r *TariffRepository) Delete(id int) error {
	result := r.DB.Delete(&models.Tariff{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete tariff: %w", result.Error)
	}
	return nil
}
