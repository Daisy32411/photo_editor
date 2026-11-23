package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"photo_editor/internal/models"
	"photo_editor/internal/service"
)

func AuthMiddleware(
	auth *service.AuthService,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		token, ok := strings.CutPrefix(header, "Bearer ")

		if !ok || strings.TrimSpace(token) == "" {
			http.Error(
				w,
				"Missing or invalid Authorization header",
				http.StatusUnauthorized,
			)
			return
		}

		_, err := auth.ValidateSession(r.Context(), token)

		if errors.Is(err, models.ErrSessionNotFound) {
			http.Error(
				w,
				"Invalid or expired session token",
				http.StatusUnauthorized,
			)
			return
		}

		if err != nil {
			log.Printf("validate session: %v", err)

			http.Error(
				w,
				"Authentication temporarily unavailable",
				http.StatusServiceUnavailable,
			)
			return
		}

		next(w, r)
	}
}
