package handlers

import (
	"net/http"
	"photo_editor/internal/service"
	"strings"
)

func AuthMiddleware(auth *service.AuthService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		token, ok := strings.CutPrefix(header, "Bearer ")
		if ok == false || token == "" {
			http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		_, valid := auth.ValidateSession(token)
		if !valid {
			http.Error(w, "Invalid or expired session token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
