package service

import "photo_editor/internal/models"

type AuthStorage interface {
	SaveUser(models.User) bool

	GetUserByUsername(username string) (models.User, bool)

	SaveSession(session models.Session)

	GetSession(token string) (models.Session, bool)
}
