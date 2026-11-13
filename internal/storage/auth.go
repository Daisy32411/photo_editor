package storage

import (
	"photo_editor/internal/models"
	"sync"
)

type AuthMemoryStorage struct {
	mu       sync.RWMutex
	users    map[string]models.User
	sessions map[string]models.Session
}

func NewAuthMemoryStorage() *AuthMemoryStorage {
	return &AuthMemoryStorage{
		users:    make(map[string]models.User),
		sessions: make(map[string]models.Session),
	}
}

func (s *AuthMemoryStorage) SaveUser(user models.User) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.users[user.Username]
	if exists {
		return false
	}

	s.users[user.Username] = user

	return true
}

func (s *AuthMemoryStorage) GetUserByUsername(username string) (models.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]

	return user, exists
}

func (s *AuthMemoryStorage) SaveSession(session models.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.SessionID] = session
}

func (s *AuthMemoryStorage) GetSession(token string) (models.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[token]

	return session, exists
}
