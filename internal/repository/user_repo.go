package repository

import (
	"database/sql"
	"errors"
	"vpn-backend/internal/db"
	"vpn-backend/internal/models"
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{DB: db.DB}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (email, password, uuid, tariff_id, created_at, is_banned)
		VALUES ($1, $2, $3, $4, NOW(), false)
		RETURNING id, created_at
	`
	err := r.DB.QueryRow(query, user.Email, user.Password, user.UUID, user.TariffID).Scan(&user.ID, &user.CreatedAt)
	return err
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, email, password, uuid, tariff_id, created_at, is_banned FROM users WHERE email = $1`
	err := r.DB.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.UUID, &user.TariffID, &user.CreatedAt, &user.IsBanned,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // пользователь не найден
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetUserByUUID(uuid string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, email, password, uuid, tariff_id, created_at, is_banned FROM users WHERE uuid = $1`
	err := r.DB.QueryRow(query, uuid).Scan(
		&user.ID, &user.Email, &user.Password, &user.UUID, &user.TariffID, &user.CreatedAt, &user.IsBanned,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateTariff(userID, tariffID int) error {
	query := `UPDATE users SET tariff_id = $1 WHERE id = $2`
	_, err := r.DB.Exec(query, tariffID, userID)
	return err
}

func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	rows, err := r.DB.Query(`SELECT id, email, password, uuid, tariff_id, created_at, is_banned FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.ID, &u.Email, &u.Password, &u.UUID, &u.TariffID, &u.CreatedAt, &u.IsBanned,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) BanUser(userID int, ban bool) error {
	query := `UPDATE users SET is_banned = $1 WHERE id = $2`
	_, err := r.DB.Exec(query, ban, userID)
	return err
}
