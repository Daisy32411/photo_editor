package service

import (
	"context"

	"photo_editor/internal/models"
)

type TaskStorage interface {
	Save(ctx context.Context, task models.Task) error

	Get(ctx context.Context, id string) (models.Task, error)

	Update(ctx context.Context, task models.Task) error
}
