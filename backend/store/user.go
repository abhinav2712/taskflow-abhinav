package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhinav2712/taskflow-abhinav/model"
)

func CreateUser(
	ctx context.Context,
	pool *pgxpool.Pool,
	name string,
	email string,
	hashedPassword string,
) (model.User, error) {
	const query = `
		INSERT INTO users (name, email, password)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password, created_at
	`

	var user model.User
	err := pool.QueryRow(ctx, query, name, email, hashedPassword).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
	)

	return user, err
}

func GetUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (model.User, error) {
	const query = `
		SELECT id, name, email, password, created_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
	)

	return user, err
}

func ListUsers(ctx context.Context, pool *pgxpool.Pool) ([]model.User, error) {
	const query = `
		SELECT id, name, email, created_at
		FROM users
		ORDER BY name ASC, email ASC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.CreatedAt,
		); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, rows.Err()
}
