package db

import (
	"context"
	"database/sql"
	"embed"
	"log"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewPool(databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func RunMigrations(databaseURL string) {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		log.Fatalf("migrations source: %v", err)
	}

	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("migrations db open: %v", err)
	}

	dbDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		log.Fatalf("migrations driver: %v", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		log.Fatalf("migrations init: %v", err)
	}
	defer migrator.Close() // also closes sqlDB internally

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrations up: %v", err)
	}

	slog.Info("database migrations complete")
}
