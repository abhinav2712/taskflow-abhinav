# TaskFlow

## 1. Overview

TaskFlow is a small full-stack task management app built for the take-home assignment.

It includes:
- Go + Chi backend
- PostgreSQL database
- React + TypeScript + Vite frontend
- JWT-based authentication
- Projects and tasks CRUD
- task filtering and optimistic status updates in the UI

The repo is structured as a simple monorepo:
- `backend/` for the Go API
- `frontend/` for the React app

## 2. Architecture Decisions

- Backend framework: `chi`
  - Keeps routing small and explicit without adding a larger framework.
- Backend structure: `handler -> store`
  - Handlers own HTTP concerns.
  - Stores own SQL/database access.
  - No service layer was added to keep the project aligned with the assignment and easy to review.
- Database migrations:
  - SQL migrations live in `backend/db/migrations/`.
  - They are embedded into the Go binary and executed on backend startup.
  - Seed data is part of the migration set rather than a separate manual script.
- Frontend routing:
  - React Router is used for navigation and protected routes.
- Frontend state:
  - Zustand with `persist` is used for auth state so JWT/user data survive refreshes.
- UI approach:
  - Custom React components + custom CSS were used.
  - No external component library was added.
  - The visual direction is a lightweight Zomato-inspired card UI with responsive layouts.

## 3. Running Locally

### Option A: Docker Compose

1. Create the root env file:

```bash
cp .env.example .env
```

2. Start the full stack:

```bash
docker compose up --build
```

Services:
- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:8080`
- Swagger docs: `http://localhost:8080/docs/`
- PostgreSQL: `localhost:5432`

Notes:
- The backend waits for Postgres health before starting.
- On backend startup, embedded migrations are applied automatically.
- The seed migration is also applied automatically.

### Option B: Run Backend + Frontend Separately

1. Create the root env file:

```bash
cp .env.example .env
```

2. Start PostgreSQL separately.

3. Install backend dependencies:

```bash
cd backend
go mod tidy
```

4. Run backend tests:

```bash
make test
```

This runs the package-level backend tests.

For integration tests in `backend/tests/`, set `TEST_DATABASE_URL` first and run:

```bash
TEST_DATABASE_URL=postgres://taskflow:taskflow_secret@localhost:5432/taskflow?sslmode=disable make test-integration
```

5. Run the backend:

```bash
make run
```

6. For hot reload during backend development:

```bash
make dev
```

`make dev` uses `air` if it is installed. If `air` is not installed, it falls back to `go run ./cmd/server`.

Install `air` once with:

```bash
go install github.com/air-verse/air@latest
```

7. For the Vite frontend, create `frontend/.env` with:

```bash
VITE_API_URL=http://localhost:8080
```

8. Run the frontend:

```bash
cd frontend
npm install
npm run dev
```

Default local URLs:
- Frontend dev server: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- Swagger docs: `http://localhost:8080/docs/`

## 4. Running Migrations

There is no separate custom migration CLI in this project.

Migration strategy:
- SQL files are stored in `backend/db/migrations/`
- the backend embeds them with `go:embed`
- the backend runs them on startup using `golang-migrate`

Current migration set:
- `000001_init` - schema
- `000002_seed` - seed data
- `000003_project_updated_at` - follow-up schema tweak

Practical usage:
- `docker compose up --build` will start Postgres and then the backend, which applies migrations automatically
- `go run ./cmd/server` also applies migrations automatically when running the backend locally

Optional manual seed files are also available in `backend/seeds/`:
- `backend/seeds/test_data.sql`
- `backend/seeds/cleanup.sql`

Those mirror the migration seed data and are useful if you want to reapply or remove the fixed reviewer dataset manually after the schema already exists.

Integration test notes:
- `backend/tests/` contains black-box HTTP integration tests
- they expect a reachable PostgreSQL database via `TEST_DATABASE_URL`
- they reuse the migrated schema and existing seeded reviewer data

If you want a clean reseed with Docker:

```bash
docker compose down -v
docker compose up --build
```

That recreates the database volume, reapplies the schema migrations, and reapplies the seed migration.

## 5. Test Credentials

Seeded reviewer account:

- Email: `test@example.com`
- Password: `password123`

Seeded data also includes:
- 1 test user
- 1 demo project
- 3 demo tasks with different statuses

Manual seed helpers:

```bash
psql "$DATABASE_URL" -f backend/seeds/test_data.sql
psql "$DATABASE_URL" -f backend/seeds/cleanup.sql
```

## 6. API Reference

All responses are JSON.

Full API documentation (Swagger) is available at `http://localhost:8080/docs/`.

### Auth

- `POST /auth/register`
- `POST /auth/login`

### Users

- `GET /users`
  - Protected
  - Returns `id`, `name`, and `email` for available users

### Projects

- `GET /projects`
  - Protected
  - Returns projects visible to the authenticated user
- `POST /projects`
  - Protected
  - Creates a project owned by the authenticated user
- `GET /projects/:id`
  - Protected
  - Returns project details and its tasks
- `PATCH /projects/:id`
  - Protected
  - Owner only
- `DELETE /projects/:id`
  - Protected
  - Owner only

### Tasks

- `GET /projects/:id/tasks`
  - Protected
  - Supports optional filters:
    - `?status=todo|in_progress|done`
    - `?assignee=<user-id>`
- `POST /projects/:id/tasks`
  - Protected
  - Creates a task inside the project
- `PATCH /tasks/:id`
  - Protected
  - Supports updating title, description, status, priority, assignee, and due date
- `DELETE /tasks/:id`
  - Protected
  - Allowed for the project owner or the task creator

### Common Error Shapes

- `400`

```json
{
  "error": "validation failed",
  "fields": {
    "name": "is required"
  }
}
```

- `401`

```json
{
  "error": "unauthorized"
}
```

- `403`

```json
{
  "error": "forbidden"
}
```

- `404`

```json
{
  "error": "not found"
}
```

## 7. What I'd Do With More Time

- Add backend tests for handlers and store queries
- Add frontend integration tests for auth, projects, and task flows
- Improve assignee filtering to support all users directly from the UI
- Add pagination and search for larger project/task lists
- Add retry actions on error states in the frontend
- Tighten Docker polish further with smaller runtime images and explicit healthchecks for the app containers
- Add a production-focused deployment note section
