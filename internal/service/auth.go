package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"photo_editor/internal/models"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidInput       = errors.New("username and password are required")
)

type AuthService struct {
	storage AuthStorage
}

func NewAuthService(store AuthStorage) *AuthService {
	return &AuthService{
		storage: store,
	}
}

func (s *AuthService) Register(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" {
		return ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
	}

	saved := s.storage.SaveUser(user)
	if !saved {
		return ErrUserExists
	}

	return nil
}

func (s *AuthService) Login(username, password string) (string, error) {
	username = strings.TrimSpace(username)
	user, exists := s.storage.GetUserByUsername(username)
	if !exists {
		return "", ErrInvalidCredentials
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)

	if err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	session := models.Session{
		UserID:    user.ID,
		SessionID: token,
	}

	s.storage.SaveSession(session)

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

func (s *AuthService) ValidateSession(token string) (models.Session, bool) {
	return s.storage.GetSession(token)
}
