package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"photo_editor/internal/models"
	"photo_editor/internal/service"
)

type contextKey string

const userIDKey contextKey = "user_id"

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

		session, err := auth.ValidateSession(r.Context(), token)

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

		ctx := context.WithValue(
			r.Context(),
			userIDKey,
			session.UserID,
		)

		next(w, r.WithContext(ctx))
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)

	return userID, ok && userID != ""
}
