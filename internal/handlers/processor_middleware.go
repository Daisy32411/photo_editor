package handlers

import (
	"crypto/subtle"
	"net/http"
)

func ProcessorMiddleware(
	token string,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get("X-Processor-Token")

		if token == "" || providedToken == "" ||
			subtle.ConstantTimeCompare(
				[]byte(providedToken),
				[]byte(token),
			) != 1 {
			http.Error(
				w,
				"Unauthorized processor",
				http.StatusUnauthorized,
			)
			return
		}

		next(w, r)
	}
}
