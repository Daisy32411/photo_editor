package handlers

import "photo_editor/internal/models"

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateImageTaskRequest struct {
	Image  string        `json:"image"`
	Filter models.Filter `json:"filter"`
}
