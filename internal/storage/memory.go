package storage

import (
	"photo_editor/internal/models"
	"sync"
)

type MemoryStorage struct {
	mu    sync.RWMutex
	tasks map[string]models.Task
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{tasks: make(map[string]models.Task)}
}

func (s *MemoryStorage) Save(task models.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
}

func (s *MemoryStorage) Get(id string) (models.Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.tasks[id]

	return value, exists
}

func (s *MemoryStorage) Update(task models.Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
}
