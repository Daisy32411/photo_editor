package handlers

import (
	"encoding/json"
	"net/http"
	"photo_editor/internal/service"
)

type TaskHandler struct {
	service *service.TaskService
}

func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{
		service: service,
	}
}

// CreateTask handles POST /task.
// @Summary Create a new task
// @Description Creates an asynchronous image processing task
// @Tags tasks
// @Produce json
// @Success 201 {object} CreateTaskResponse "Task created successfully"
// @Router /task [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	id := h.service.CreateTask()

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(CreateTaskResponse{
		TaskID: id,
	})
}

// GetStatus handles GET /status/{id}.
// @Summary Get task status
// @Description Returns the current processing status of a task
// @Tags tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} GetStatusResponse "Task status"
// @Failure 404 {string} string "Task not found"
// @Router /status/{id} [get]
func (h *TaskHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	task, exists := h.service.GetTask(id)
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(GetStatusResponse{
		Status: task.Status,
	})
}

// GetResult handles GET /result/{id}.
// @Summary Get task result
// @Description Returns the result of a completed task
// @Tags tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} GetResultResponse "Task result"
// @Failure 404 {string} string "Task not found"
// @Failure 409 {string} string "Task is not ready"
// @Router /result/{id} [get]
func (h *TaskHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	task, exists := h.service.GetTask(id)
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if task.Status != "ready" {
		http.Error(w, "Task is not ready", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(GetResultResponse{
		Result: task.Result,
	})
}
