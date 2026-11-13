package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"photo_editor/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: svc,
	}
}

// Register handles POST /register.
// @Summary Register a new user
// @Description Registers a new user with a username and password
// @Tags auth
// @Accept json
// @Param credentials body AuthRequest true "User registration credentials"
// @Success 201 {object} nil "User registered successfully (empty response body)"
// @Failure 400 {string} string "Invalid JSON or missing credentials"
// @Failure 409 {string} string "User already exists"
// @Failure 500 {string} string "Internal server error"
// @Router /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err = h.service.Register(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			http.Error(w, "User already exists", http.StatusConflict)

		case errors.Is(err, service.ErrInvalidInput):
			http.Error(w, "Username and password are required", http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Login handles POST /login.
// @Summary Authenticate a user
// @Description Authenticates a user and creates a new session
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body AuthRequest true "User login credentials"
// @Success 200 {object} LoginResponse "Authentication successful"
// @Failure 400 {string} string "Invalid JSON"
// @Failure 401 {string} string "Invalid username or password"
// @Failure 500 {string} string "Internal server error"
// @Router /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)

		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(LoginResponse{
		Token: token,
	})
}
