package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/abhinav2712/taskflow-abhinav/config"
	"github.com/abhinav2712/taskflow-abhinav/db"
	"github.com/abhinav2712/taskflow-abhinav/handler"
	"github.com/abhinav2712/taskflow-abhinav/middleware"
)

func main() {
	cfg := config.Load()

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	db.RunMigrations(cfg.DatabaseURL)

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))
	router.Use(middleware.Logger)

	// Public auth routes — no auth middleware
	router.Post("/auth/register", handler.Register(pool, cfg.JWTSecret))
	router.Post("/auth/login", handler.Login(pool, cfg.JWTSecret))

	router.Route("/projects", func(r chi.Router) {
		r.Use(middleware.Authenticate(cfg.JWTSecret))
		r.Get("/", handler.ListProjects(pool))
		r.Post("/", handler.CreateProject(pool))
		r.Get("/{id}", handler.GetProject(pool))
		r.Get("/{id}/tasks", handler.ListTasks(pool))
		r.Post("/{id}/tasks", handler.CreateTask(pool))
		r.Patch("/{id}", handler.UpdateProject(pool))
		r.Delete("/{id}", handler.DeleteProject(pool))
	})

	router.Route("/tasks", func(r chi.Router) {
		r.Use(middleware.Authenticate(cfg.JWTSecret))
		r.Patch("/{id}", handler.UpdateTask(pool))
		r.Delete("/{id}", handler.DeleteTask(pool))
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine so we can listen for shutdown signal
	go func() {
		slog.Info("TaskFlow API starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Block until SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}

	slog.Info("server stopped")
}
