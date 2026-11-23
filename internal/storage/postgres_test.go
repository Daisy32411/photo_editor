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

func TestPostgresTaskStorage(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_URL")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_URL is not set")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL config: %v", err)
	}
	if !strings.HasSuffix(strings.ToLower(config.ConnConfig.Database), "_test") {
		t.Fatal("POSTGRES_TEST_URL must point to a dedicated database whose name ends with _test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}

	var tableExists bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.tasks') IS NOT NULL").Scan(&tableExists); err != nil {
		t.Fatalf("check tasks table: %v", err)
	}
	if !tableExists {
		t.Fatal("tasks table does not exist in the test database; apply migrations first")
	}

	store := NewPostgresTaskStorage(pool)
	task := models.Task{
		ID:     uuid.NewString(),
		Status: "in_progress",
		Result: "",
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, "DELETE FROM tasks WHERE id = $1", task.ID); err != nil {
			t.Errorf("delete test task: %v", err)
		}
	})

	if err := store.Save(ctx, task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Run("Get", func(t *testing.T) {
		got, err := store.Get(ctx, task.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got != task {
			t.Fatalf("Get: got %+v, want %+v", got, task)
		}
	})

	t.Run("Update", func(t *testing.T) {
		updated := task
		updated.Status = "ready"
		updated.Result = "test-result"

		if err := store.Update(ctx, updated); err != nil {
			t.Fatalf("Update: %v", err)
		}

		got, err := store.Get(ctx, task.ID)
		if err != nil {
			t.Fatalf("Get after Update: %v", err)
		}
		if got != updated {
			t.Fatalf("Get after Update: got %+v, want %+v", got, updated)
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		_, err := store.Get(ctx, uuid.NewString())
		if !errors.Is(err, models.ErrTaskNotFound) {
			t.Fatalf("Get: got error %v, want ErrTaskNotFound", err)
		}
	})

	t.Run("UpdateNotFound", func(t *testing.T) {
		missing := models.Task{
			ID:     uuid.NewString(),
			Status: "ready",
			Result: "test-result",
		}
		if err := store.Update(ctx, missing); !errors.Is(err, models.ErrTaskNotFound) {
			t.Fatalf("Update: got error %v, want ErrTaskNotFound", err)
		}
	})

	t.Run("SaveDuplicate", func(t *testing.T) {
		if err := store.Save(ctx, task); err == nil {
			t.Fatal("Save: expected error for duplicate task ID")
		}
	})
}
