package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"photo_editor/internal/models"

	"github.com/google/uuid"
)

const MaxTaskMessageSize = 99 * 1024 * 1024

var ErrTaskTooLarge = errors.New("image task is too large")

type TaskService struct {
	storage   TaskStorage
	publisher TaskPublisher
}

func NewTaskService(
	store TaskStorage,
	publisher TaskPublisher,
) *TaskService {
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

	task := models.Task{
		ID:     id,
		Status: "in_progress",
		Result: "",
	}

	message := models.ImageTaskMessage{
		TaskID: id,
		Image:  image,
		Filter: filter,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("marshal task message: %w", err)
	}

	if len(body) > MaxTaskMessageSize {
		return "", ErrTaskTooLarge
	}

	if err := s.storage.Save(ctx, task); err != nil {
		return "", fmt.Errorf("save task: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(
		ctx,
		30*time.Second,
	)
	defer cancel()

	err = s.publisher.Publish(publishCtx, body)
	if err != nil {
		task.Status = "failed"

		updateCtx, updateCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer updateCancel()

		updateErr := s.storage.Update(updateCtx, task)
		if updateErr != nil {
			return "", fmt.Errorf(
				"publish task: %w; mark task as failed: %v",
				err,
				updateErr,
			)
		}

		return "", fmt.Errorf("publish task: %w", err)
	}

	return id, nil
}

func (s *TaskService) CommitTask(
	ctx context.Context,
	id string,
	status string,
	result string,
) error {
	task, err := s.storage.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("commit task: %w", err)
	}

	task.Status = status

	if status == "ready" {
		task.Result = result
	} else {
		task.Result = ""
	}

	if err := s.storage.Update(ctx, task); err != nil {
		return fmt.Errorf("commit task update: %w", err)
	}

	return nil
}

func (s *TaskService) GetTask(
	ctx context.Context,
	id string,
) (models.Task, error) {
	return s.storage.Get(ctx, id)
}
