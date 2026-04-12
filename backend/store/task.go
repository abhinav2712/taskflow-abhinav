package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhinav2712/taskflow-abhinav/model"
)

type CreateTaskInput struct {
	Title       string
	Description *string
	Priority    string
	AssigneeID  *uuid.UUID
	DueDate     *string
}

type UpdateTaskInput struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *string
	AssigneeID  *uuid.UUID
	DueDate     *string
}

func ListTasks(
	ctx context.Context,
	pool *pgxpool.Pool,
	projectID uuid.UUID,
	status *string,
	assigneeID *uuid.UUID,
) ([]model.Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id, creator_id, due_date::text, created_at, updated_at
		FROM tasks
		WHERE project_id = $1
	`

	args := []any{projectID}
	argPos := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	if assigneeID != nil {
		query += fmt.Sprintf(" AND assignee_id = $%d", argPos)
		args = append(args, *assigneeID)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]model.Task, 0)
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.Priority,
			&task.ProjectID,
			&task.AssigneeID,
			&task.CreatorID,
			&task.DueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func CreateTask(
	ctx context.Context,
	pool *pgxpool.Pool,
	projectID uuid.UUID,
	creatorID uuid.UUID,
	input CreateTaskInput,
) (model.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, priority, project_id, assignee_id, creator_id, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, status, priority, project_id, assignee_id, creator_id, due_date::text, created_at, updated_at
	`

	var task model.Task
	err := pool.QueryRow(
		ctx,
		query,
		input.Title,
		input.Description,
		input.Priority,
		projectID,
		input.AssigneeID,
		creatorID,
		input.DueDate,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&task.ProjectID,
		&task.AssigneeID,
		&task.CreatorID,
		&task.DueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	return task, err
}

func GetTask(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (model.Task, error) {
	const query = `
		SELECT id, title, description, status, priority, project_id, assignee_id, creator_id, due_date::text, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	var task model.Task
	err := pool.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&task.ProjectID,
		&task.AssigneeID,
		&task.CreatorID,
		&task.DueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	return task, err
}

func UpdateTask(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	input UpdateTaskInput,
) (model.Task, error) {
	setClauses := make([]string, 0, 7)
	args := make([]any, 0, 7)
	argPos := 1

	if input.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argPos))
		args = append(args, *input.Title)
		argPos++
	}

	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *input.Description)
		argPos++
	}

	if input.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *input.Status)
		argPos++
	}

	if input.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argPos))
		args = append(args, *input.Priority)
		argPos++
	}

	if input.AssigneeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argPos))
		args = append(args, *input.AssigneeID)
		argPos++
	}

	if input.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argPos))
		args = append(args, *input.DueDate)
		argPos++
	}

	setClauses = append(setClauses, "updated_at = now()")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE tasks
		SET %s
		WHERE id = $%d
		RETURNING id, title, description, status, priority, project_id, assignee_id, creator_id, due_date::text, created_at, updated_at
	`, strings.Join(setClauses, ", "), argPos)

	var task model.Task
	err := pool.QueryRow(ctx, query, args...).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&task.ProjectID,
		&task.AssigneeID,
		&task.CreatorID,
		&task.DueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	return task, err
}

func DeleteTask(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	const query = `DELETE FROM tasks WHERE id = $1`
	_, err := pool.Exec(ctx, query, id)
	return err
}
