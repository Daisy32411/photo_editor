package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTaskBodyLimit(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "OversizedJSON",
			body: `{"image":"` +
				strings.Repeat("a", 100) +
				`","filter":{"name":"Negative"}}`,
		},
		{
			name: "OversizedTrailingWhitespace",
			body: `{"image":"abc","filter":{"name":"Negative"}}` +
				strings.Repeat(" ", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewTaskHandler(nil, 64)

			request := httptest.NewRequest(
				http.MethodPost,
				"/task",
				strings.NewReader(tt.body),
			)

			ctx := context.WithValue(
				request.Context(),
				userIDKey,
				"test-user",
			)

			request = request.WithContext(ctx)

			recorder := httptest.NewRecorder()

			handler.CreateTask(recorder, request)

			if recorder.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf(
					"got status %d, want %d; body: %s",
					recorder.Code,
					http.StatusRequestEntityTooLarge,
					recorder.Body.String(),
				)
			}
		})
	}
}
