package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"photo_editor/internal/models"
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
// @Security BearerAuth
// @Success 201 {object} CreateTaskResponse "Task created successfully"
// @Failure 401 {string} string "Unauthorized"
// @Router /task [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateImageTaskRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Image == "" {
		http.Error(w, "Image is required", http.StatusBadRequest)
		return
	}

	if req.Filter.Name == "" {
		http.Error(w, "Filter name is required", http.StatusBadRequest)
		return
	}

	id, err := h.service.CreateTask(r.Context(), req.Image, req.Filter)
	if err != nil {
		log.Printf("Failed to create task: %v", err)

		if errors.Is(err, service.ErrTaskTooLarge) {
			http.Error(
				w,
				"Image is too large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}

		http.Error(
			w,
			"Failed to enqueue image processing task",
			http.StatusServiceUnavailable,
		)
		return
	}

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
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} GetStatusResponse "Task status"
// @Failure 401 {string} string "Unauthorized"
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
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Success 200 {object} GetResultResponse "Task result"
// @Failure 401 {string} string "Unauthorized"
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

	imageBytes, err := base64.StdEncoding.DecodeString(task.Result)
	if err != nil {
		http.Error(w, "Failed to decode task result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")

	w.Write(imageBytes)
}

// CommitResult принимает результат обработки от ImageProcessor
func (h *TaskHandler) CommitResult(w http.ResponseWriter, r *http.Request) {
	var req models.CommitRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.TaskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}
	if req.Status != "ready" && req.Status != "failed" {
		http.Error(w, "Invalid task status", http.StatusBadRequest)
		return
	}
	if req.Status == "ready" && req.Result == "" {
		http.Error(w, "Result is required for completed tasks", http.StatusBadRequest)
		return
	}

	exists := h.service.CommitTask(req.TaskID, req.Status, req.Result)
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
