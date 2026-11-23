package service

import (
	"context"

	"photo_editor/internal/models"
)

type UserStorage interface {
	SaveUser(ctx context.Context, user models.User) error
	GetUserByUsername(ctx context.Context, username string) (models.User, error)
}

type SessionStorage interface {
	SaveSession(ctx context.Context, session models.Session) error
	GetSession(ctx context.Context, token string) (models.Session, error)
}
