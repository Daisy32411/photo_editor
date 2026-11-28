package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"photo_editor/internal/models"

	"github.com/redis/go-redis/v9"
)

const SessionTTL = 7 * 24 * time.Hour

type RedisSessionStorage struct {
	client *redis.Client
}

func NewRedisSessionStorage(client *redis.Client) *RedisSessionStorage {
	return &RedisSessionStorage{
		client: client,
	}
}

func (s *RedisSessionStorage) SaveSession(
	ctx context.Context,
	session models.Session,
) error {
	key := "session:" + session.SessionID

	err := s.client.Set(
		ctx,
		key,
		session.UserID,
		SessionTTL,
	).Err()

	if err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return nil
}

func (s *RedisSessionStorage) GetSession(
	ctx context.Context,
	token string,
) (models.Session, error) {
	key := "session:" + token

	userID, err := s.client.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return models.Session{}, models.ErrSessionNotFound
	}

	if err != nil {
		return models.Session{}, fmt.Errorf("get session: %w", err)
	}

	return models.Session{
		UserID:    userID,
		SessionID: token,
	}, nil
}
