package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"photo_editor/internal/models"
	"photo_editor/internal/service"
)

type TaskHandler struct {
	service     *service.TaskService
	maxBodySize int64
}

func NewTaskHandler(
	service *service.TaskService,
	maxBodySize int64,
) *TaskHandler {
	return &TaskHandler{
		service:     service,
		maxBodySize: maxBodySize,
	}
}

// CreateTask handles POST /task.
// @Summary Create an image processing task
// @Description Creates an asynchronous image processing task using RabbitMQ.
// @Description The image must be a Base64-encoded PNG.
// @Description Supported filters: Negative, Blur, Sharpen, FlipX.
// @Description Blur and Sharpen require parameters.sigma greater than 0 and not greater than 20.
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateImageTaskRequest true "Image and filter configuration"
// @Success 201 {object} CreateTaskResponse "Task successfully created"
// @Failure 400 {string} string "Invalid JSON, missing image or filter"
// @Failure 401 {string} string "Unauthorized"
// @Failure 413 {string} string "Image task is too large"
// @Failure 503 {string} string "Failed to publish task to RabbitMQ"
// @Router /task [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBodySize)
	defer r.Body.Close()

	var req CreateImageTaskRequest

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&req)

	if err == nil {
		err = decoder.Decode(new(any))

		if err == io.EOF {
			err = nil
		} else if err == nil {
			err = errors.New("unexpected data after JSON")
		}
	}

	if err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			http.Error(
				w,
				"Request body is too large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}

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

	id, err := h.service.CreateTask(
		r.Context(),
		userID,
		req.Image,
		req.Filter,
	)
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
// @Summary Get image processing task status
// @Description Returns the current task status: in_progress, ready, or failed.
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Task ID (UUID)"
// @Success 200 {object} GetStatusResponse "Current task status"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Task not found"
// @Failure 400 {string} string "Invalid task ID"
// @Router /status/{id} [get]
func (h *TaskHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.service.GetTask(r.Context(), id.String(), userID)
	if err != nil {
		writeTaskError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(GetStatusResponse{
		Status: task.Status,
	})
}

// GetResult handles GET /result/{id}.
// @Summary Download processed image
// @Description Returns the processed PNG image when the task status is ready.
// @Description Returns HTTP 409 if the task is still processing or has failed.
// @Tags tasks
// @Produce png
// @Security BearerAuth
// @Param id path string true "Task ID (UUID)"
// @Success 200 {file} file "Processed PNG image"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Task not found"
// @Failure 409 {string} string "Task is not ready"
// @Failure 500 {string} string "Failed to decode task result"
// @Failure 400 {string} string "Invalid task ID"
// @Router /result/{id} [get]
func (h *TaskHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.service.GetTask(r.Context(), id.String(), userID)
	if err != nil {
		writeTaskError(w, err)
		return
	}

	if task.Status != "ready" {
		http.Error(w, "Task is not ready", http.StatusConflict)
		return
	}

	imageBytes, err := base64.StdEncoding.DecodeString(task.Result)
	if err != nil {
		writeTaskError(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")

	w.Write(imageBytes)
}

// CommitResult handles POST /commit.
// @Summary Commit image processing result
// @Description Internal endpoint used by ImageProcessor to report task completion.
// @Description Status must be ready or failed.
// @Description If status is ready, result must contain a Base64-encoded PNG.
// @Tags internal
// @Accept json
// @Param request body models.CommitRequest true "Task ID, status and processing result"
// @Success 204 "Result successfully committed"
// @Failure 400 {string} string "Invalid request body or task status"
// @Failure 404 {string} string "Task not found"
// @Failure 401 {string} string "Unauthorized processor"
// @Failure 409 {string} string "Task is already completed"
// @Router /commit [post]
// @Security ProcessorAuth
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
	if _, err := uuid.Parse(req.TaskID); err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
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

	err = h.service.CommitTask(r.Context(), req.TaskID, req.Status, req.Result)
	if err != nil {
		writeTaskError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeTaskError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrTaskNotFound) {
		http.Error(
			w,
			"Task not found",
			http.StatusNotFound,
		)
		return
	}

	if errors.Is(err, models.ErrTaskConflict) {
		http.Error(
			w,
			"Task is already completed",
			http.StatusConflict,
		)
		return
	}

	log.Printf("Task storage error: %v", err)

	http.Error(
		w,
		"Internal server error",
		http.StatusInternalServerError,
	)
}
