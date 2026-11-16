package service

import (
	"context"
	"encoding/json"
	"errors"
	"photo_editor/internal/models"
	"time"

	"github.com/google/uuid"
)

const MaxTaskMessageSize = 99 * 1024 * 1024

var ErrTaskTooLarge = errors.New("image task is too large")

type TaskService struct {
	storage   TaskStorage
	publisher TaskPublisher
}

func NewTaskService(store TaskStorage, publisher TaskPublisher) *TaskService {
	return &TaskService{
		storage:   store,
		publisher: publisher,
	}
}

func (s *TaskService) CreateTask(
	ctx context.Context,
	image string,
	filter models.Filter,
) (string, error) {
	id := uuid.NewString()

	var task models.Task

	task.ID = id
	task.Status = "in_progress"
	task.Result = ""

	message := models.ImageTaskMessage{
		TaskID: id,
		Image:  image,
		Filter: filter,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	if len(body) > MaxTaskMessageSize {
		return "", ErrTaskTooLarge
	}

	s.storage.Save(task)

	publishCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = s.publisher.Publish(publishCtx, body)
	if err != nil {
		task.Status = "failed"
		s.storage.Update(task)
		return "", err
	}

	return id, nil
}

// CommitTask сохраняет результат обработанной задачи
func (s *TaskService) CommitTask(id, status, result string) bool {
	task, exists := s.storage.Get(id)
	if !exists {
		return false
	}

	task.Status = status
	if status == "ready" {
		task.Result = result
	} else {
		task.Result = ""
	}

	s.storage.Update(task)

	return true
}

func (s *TaskService) GetTask(id string) (models.Task, bool) {
	return s.storage.Get(id)
}
