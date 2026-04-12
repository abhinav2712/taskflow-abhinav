package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhinav2712/taskflow-abhinav/db"
	"github.com/abhinav2712/taskflow-abhinav/handler"
	"github.com/abhinav2712/taskflow-abhinav/middleware"
)

const testJWTSecret = "integration-test-secret"

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

type projectsResponse struct {
	Projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"projects"`
}

func TestRegisterAndLoginFlow(t *testing.T) {
	server, pool := newIntegrationServer(t)

	email := fmt.Sprintf("integration-%d@example.com", time.Now().UnixNano())
	password := "password123"
	name := "Integration User"

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE email = $1", email)
	})

	registerPayload := map[string]string{
		"name":     name,
		"email":    email,
		"password": password,
	}

	registerResponse := postJSON(t, server.URL+"/auth/register", registerPayload, "")
	defer registerResponse.Body.Close()

	if registerResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected register status %d, got %d", http.StatusCreated, registerResponse.StatusCode)
	}

	var registered authResponse
	if err := json.NewDecoder(registerResponse.Body).Decode(&registered); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}

	if registered.Token == "" {
		t.Fatal("expected register response to include token")
	}

	if registered.User.Email != email {
		t.Fatalf("expected registered email %q, got %q", email, registered.User.Email)
	}

	loginPayload := map[string]string{
		"email":    email,
		"password": password,
	}

	loginResponse := postJSON(t, server.URL+"/auth/login", loginPayload, "")
	defer loginResponse.Body.Close()

	if loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginResponse.StatusCode)
	}

	var loggedIn authResponse
	if err := json.NewDecoder(loginResponse.Body).Decode(&loggedIn); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	if loggedIn.Token == "" {
		t.Fatal("expected login response to include token")
	}
}

func TestProtectedProjectsRouteRequiresAuth(t *testing.T) {
	server, _ := newIntegrationServer(t)

	request, err := http.NewRequest(http.MethodGet, server.URL+"/projects", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.StatusCode)
	}
}

func TestSeededUserCanListProjects(t *testing.T) {
	server, _ := newIntegrationServer(t)

	loginResponse := postJSON(t, server.URL+"/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}, "")
	defer loginResponse.Body.Close()

	if loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected seeded login status %d, got %d", http.StatusOK, loginResponse.StatusCode)
	}

	var authPayload authResponse
	if err := json.NewDecoder(loginResponse.Body).Decode(&authPayload); err != nil {
		t.Fatalf("failed to decode seeded login response: %v", err)
	}

	request, err := http.NewRequest(http.MethodGet, server.URL+"/projects", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+authPayload.Token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	var projectsPayload projectsResponse
	if err := json.NewDecoder(response.Body).Decode(&projectsPayload); err != nil {
		t.Fatalf("failed to decode projects response: %v", err)
	}

	if len(projectsPayload.Projects) == 0 {
		t.Fatal("expected seeded user to see at least one project")
	}

	if projectsPayload.Projects[0].Name == "" {
		t.Fatal("expected project name to be populated")
	}
}

func newIntegrationServer(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run backend integration tests")
	}

	db.RunMigrations(databaseURL)

	pool, err := db.NewPool(databaseURL)
	if err != nil {
		t.Fatalf("failed to connect test database: %v", err)
	}
	t.Cleanup(pool.Close)

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))
	router.Use(middleware.Logger)

	router.Post("/auth/register", handler.Register(pool, testJWTSecret))
	router.Post("/auth/login", handler.Login(pool, testJWTSecret))

	router.Route("/projects", func(r chi.Router) {
		r.Use(middleware.Authenticate(testJWTSecret))
		r.Get("/", handler.ListProjects(pool))
		r.Post("/", handler.CreateProject(pool))
		r.Get("/{id}", handler.GetProject(pool))
		r.Get("/{id}/tasks", handler.ListTasks(pool))
		r.Post("/{id}/tasks", handler.CreateTask(pool))
		r.Patch("/{id}", handler.UpdateProject(pool))
		r.Delete("/{id}", handler.DeleteProject(pool))
	})

	router.Route("/tasks", func(r chi.Router) {
		r.Use(middleware.Authenticate(testJWTSecret))
		r.Patch("/{id}", handler.UpdateTask(pool))
		r.Delete("/{id}", handler.DeleteTask(pool))
	})

	router.Route("/users", func(r chi.Router) {
		r.Use(middleware.Authenticate(testJWTSecret))
		r.Get("/", handler.ListUsers(pool))
	})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, pool
}

func postJSON(t *testing.T, url string, payload any, bearerToken string) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return response
}
