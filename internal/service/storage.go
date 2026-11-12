package service

import "photo_editor/internal/models"

type TaskStorage interface {
	Save(task models.Task)

	Get(id string) (models.Task, bool)

	Update(task models.Task)
}
