package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"photo_editor/internal/models"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidInput       = errors.New("username and password are required")
)

type AuthService struct {
	users    UserStorage
	sessions SessionStorage
}

func NewAuthService(
	users UserStorage,
	sessions SessionStorage,
) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
	}
}

func (s *AuthService) Register(
	ctx context.Context,
	username string,
	password string,
) error {
	username = strings.TrimSpace(username)

	if username == "" || strings.TrimSpace(password) == "" {
		return ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := models.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
	}

	if err := s.users.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("register user: %w", err)
	}

	return nil
}

func (s *AuthService) Login(
	ctx context.Context,
	username string,
	password string,
) (string, error) {
	username = strings.TrimSpace(username)

	if username == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	user, err := s.users.GetUserByUsername(ctx, username)

	if errors.Is(err, models.ErrUserNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", fmt.Errorf("get login user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)

	if err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := generateSessionToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	session := models.Session{
		UserID:    user.ID,
		SessionID: token,
	}

	if err := s.sessions.SaveSession(ctx, session); err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}

	return token, nil
}

func generateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)

	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(tokenBytes), nil
}

func (s *AuthService) ValidateSession(
	ctx context.Context,
	token string,
) (models.Session, error) {
	return s.sessions.GetSession(ctx, token)
}
