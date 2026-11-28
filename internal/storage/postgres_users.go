package storage

import (
	"context"
	"errors"
	"fmt"

	"photo_editor/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserStorage struct {
	db *pgxpool.Pool
}

func NewPostgresUserStorage(db *pgxpool.Pool) *PostgresUserStorage {
	return &PostgresUserStorage{db: db}
}

func (s *PostgresUserStorage) SaveUser(ctx context.Context, user models.User) error {
	query := `
		INSERT INTO users (id, username, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (username) DO NOTHING
	`

	commandTag, err := s.db.Exec(ctx, query, user.ID, user.Username, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrUserExists
	}

	return nil
}

func (s *PostgresUserStorage) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	query := `
		SELECT id, username, password_hash
		FROM users
		WHERE username = $1
	`

	var user models.User

	err := s.db.QueryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, models.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("get user by username: %w", err)
	}

	return user, nil
}
