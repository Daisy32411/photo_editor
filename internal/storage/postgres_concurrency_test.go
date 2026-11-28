package storage

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"photo_editor/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentTaskUpdate(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_URL")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_URL is not set")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(
		strings.ToLower(config.ConnConfig.Database),
		"_test",
	) {
		t.Fatal("POSTGRES_TEST_URL must point to a test database")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	store := NewPostgresTaskStorage(pool)
	userStore := NewPostgresUserStorage(pool)

	user := models.User{
		ID:           uuid.NewString(),
		Username:     "test_" + uuid.NewString(),
		PasswordHash: "test-hash",
	}

	if err := userStore.SaveUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	task := models.Task{
		ID:     uuid.NewString(),
		UserID: user.ID,
		Status: "in_progress",
		Result: "",
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		if _, err := pool.Exec(
			cleanupCtx,
			"DELETE FROM tasks WHERE id = $1",
			task.ID,
		); err != nil {
			t.Errorf("delete task: %v", err)
		}

		if _, err := pool.Exec(
			cleanupCtx,
			"DELETE FROM users WHERE id = $1",
			user.ID,
		); err != nil {
			t.Errorf("delete user: %v", err)
		}
	})

	if err := store.Save(ctx, task); err != nil {
		t.Fatal(err)
	}

	first := task
	first.Status = "ready"
	first.Result = "first-result"

	second := task
	second.Status = "ready"
	second.Result = "second-result"

	type updateResult struct {
		task models.Task
		err  error
	}

	start := make(chan struct{})
	results := make(chan updateResult, 2)

	for _, candidate := range []models.Task{first, second} {
		go func(candidate models.Task) {
			<-start

			err := store.Update(ctx, candidate)

			results <- updateResult{
				task: candidate,
				err:  err,
			}
		}(candidate)
	}

	close(start)

	successCount := 0
	conflictCount := 0

	var winner models.Task

	for i := 0; i < 2; i++ {
		result := <-results

		switch {
		case result.err == nil:
			successCount++
			winner = result.task

		case errors.Is(result.err, models.ErrTaskConflict):
			conflictCount++

		default:
			t.Errorf("unexpected update error: %v", result.err)
		}
	}

	if successCount != 1 {
		t.Fatalf("expected 1 successful update, got %d", successCount)
	}

	if conflictCount != 1 {
		t.Fatalf("expected 1 conflict, got %d", conflictCount)
	}

	got, err := store.Get(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got != winner {
		t.Fatalf("got %+v, want %+v", got, winner)
	}
}
