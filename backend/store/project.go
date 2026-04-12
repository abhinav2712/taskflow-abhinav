package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhinav2712/taskflow-abhinav/model"
)

func CreateProject(
	ctx context.Context,
	pool *pgxpool.Pool,
	name string,
	description *string,
	ownerID uuid.UUID,
) (model.Project, error) {
	const query = `
		INSERT INTO projects (name, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, owner_id, created_at, updated_at
	`

	var project model.Project
	err := pool.QueryRow(ctx, query, name, description, ownerID).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.OwnerID,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	return project, err
}

func ListProjects(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.Project, error) {
	const query = `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at, p.updated_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]model.Project, 0)
	for rows.Next() {
		var project model.Project
		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.OwnerID,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, err
		}

		projects = append(projects, project)
	}

	return projects, rows.Err()
}

func GetProject(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (model.Project, error) {
	const projectQuery = `
		SELECT id, name, description, owner_id, created_at, updated_at
		FROM projects
		WHERE id = $1
	`

	var project model.Project
	err := pool.QueryRow(ctx, projectQuery, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.OwnerID,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return model.Project{}, err
	}

	const tasksQuery = `
		SELECT id, title, description, status, priority, project_id, assignee_id, creator_id, due_date::text, created_at, updated_at
		FROM tasks
		WHERE project_id = $1
		ORDER BY created_at DESC
	`

	rows, err := pool.Query(ctx, tasksQuery, id)
	if err != nil {
		return model.Project{}, err
	}
	defer rows.Close()

	project.Tasks = make([]model.Task, 0)
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
			return model.Project{}, err
		}

		project.Tasks = append(project.Tasks, task)
	}

	return project, rows.Err()
}

func UpdateProject(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	name *string,
	description *string,
) (model.Project, error) {
	setClauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	argPos := 1

	if name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *name)
		argPos++
	}

	if description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *description)
		argPos++
	}

	setClauses = append(setClauses, "updated_at = now()")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE projects
		SET %s
		WHERE id = $%d
		RETURNING id, name, description, owner_id, created_at, updated_at
	`, strings.Join(setClauses, ", "), argPos)

	var project model.Project
	err := pool.QueryRow(ctx, query, args...).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.OwnerID,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	return project, err
}

func DeleteProject(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	const query = `DELETE FROM projects WHERE id = $1`
	_, err := pool.Exec(ctx, query, id)
	return err
}
