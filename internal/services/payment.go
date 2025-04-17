package services

import "vpn-backend/internal/repository"

type PaymentService struct {
	Repo *repository.UserRepository
}

func NewPaymentService(repo *repository.UserRepository) *PaymentService {
	return &PaymentService{Repo: repo}
}

func (p *PaymentService) ChangeTariff(userID int, newTariffID int) error {
	return p.Repo.UpdateTariff(userID, newTariffID)
}
