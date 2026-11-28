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

var ErrTaskTooLarge = errors.New("image task is too large")

type TaskService struct {
	storage        TaskStorage
	publisher      TaskPublisher
	maxMessageSize int
}

func NewTaskService(
	store TaskStorage,
	publisher TaskPublisher,
	maxMessageSize int,
) *TaskService {
	return &TaskService{
		storage:        store,
		publisher:      publisher,
		maxMessageSize: maxMessageSize,
	}
}

func (s *TaskService) CreateTask(
	ctx context.Context,
	userID string,
	image string,
	filter models.Filter,
) (string, error) {

	id := uuid.NewString()

	task := models.Task{
		ID:     id,
		UserID: userID,
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

	if len(body) > s.maxMessageSize {
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
	if status != "ready" && status != "failed" {
		return fmt.Errorf("invalid task status: %s", status)
	}

	if status == "failed" {
		result = ""
	}

	task := models.Task{
		ID:     id,
		Status: status,
		Result: result,
	}

	if err := s.storage.Update(ctx, task); err != nil {
		return fmt.Errorf("commit task: %w", err)
	}

	return nil
}

func (s *TaskService) GetTask(
	ctx context.Context,
	id string,
	userID string,
) (models.Task, error) {
	return s.storage.GetByUser(ctx, id, userID)
}
