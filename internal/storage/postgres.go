package storage

import (
	"context"
	"errors"
	"fmt"
	"photo_editor/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTaskStorage struct {
	db *pgxpool.Pool
}

func NewPostgresTaskStorage(db *pgxpool.Pool) *PostgresTaskStorage {
	return &PostgresTaskStorage{
		db: db,
	}
}

func (s *PostgresTaskStorage) Save(
	ctx context.Context,
	task models.Task,
) error {
	query := `
		INSERT INTO tasks (id, status, result)
		VALUES ($1, $2, $3)
	`

	_, err := s.db.Exec(ctx, query, task.ID, task.Status, task.Result)
	if err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	return nil
}

func (s *PostgresTaskStorage) Get(
	ctx context.Context,
	id string,
) (models.Task, error) {
	query := `
		SELECT 
			id,
			status,
			result
		FROM
			tasks
		WHERE
			id = $1
	`
	var task models.Task

	err := s.db.QueryRow(ctx, query, id).
		Scan(&task.ID, &task.Status, &task.Result)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.Task{}, models.ErrTaskNotFound
	}

	if err != nil {
		return models.Task{}, fmt.Errorf("get task: %w", err)
	}

	return task, nil
}

func (s *PostgresTaskStorage) Update(
	ctx context.Context,
	task models.Task,
) error {
	query := `
		UPDATE tasks
		SET status = $1, result = $2
		WHERE id = $3
	`
	commandTag, err := s.db.Exec(ctx, query, task.Status, task.Result, task.ID)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return models.ErrTaskNotFound
	}

	return nil
}
