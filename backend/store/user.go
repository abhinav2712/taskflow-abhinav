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
