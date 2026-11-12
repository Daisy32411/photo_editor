package service

import (
	"photo_editor/internal/models"
	"time"

	"github.com/google/uuid"
)

type TaskService struct {
	storage TaskStorage
}

func NewTaskService(store TaskStorage) *TaskService {
	return &TaskService{
		storage: store,
	}
}

func (s *TaskService) CreateTask() string {
	id := uuid.NewString()

	var task models.Task

	task.ID = id
	task.Status = "in_progress"
	task.Result = ""

	s.storage.Save(task)

	go s.processTask(id)

	return id
}

func (s *TaskService) processTask(id string) {
	task, exists := s.GetTask(id)
	if !exists {
		return
	}

	time.Sleep(5 * time.Second)

	task.Status = "ready"
	task.Result = "Image processed successfully"

	s.storage.Update(task)
}

func (s *TaskService) GetTask(id string) (models.Task, bool) {
	return s.storage.Get(id)
}
