<div align="center">

<img width="1920" alt="TaskFlow desktop UI" src="https://github.com/user-attachments/assets/c5cb25a1-0def-4c4f-854f-b0029051edf4" />

<h1>TaskFlow</h1>

<p>A minimal, production-quality task management app — built as a take-home assignment.</p>

<p>
  <img src="https://img.shields.io/badge/Go-1.22-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React" />
  <img src="https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/JWT-Auth-000000?style=flat-square&logo=jsonwebtokens&logoColor=white" alt="JWT" />
</p>

</div>

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture Decisions](#2-architecture-decisions)
3. [Project Structure](#3-project-structure)
4. [Running Locally](#4-running-locally)
5. [Running Migrations](#5-running-migrations)
6. [Test Credentials](#6-test-credentials)
7. [API Reference](#7-api-reference)
8. [What I'd Do With More Time](#8-what-id-do-with-more-time)
9. [AI Assistance Disclosure](#9-ai-assistance-disclosure)

---

## 1. Overview

TaskFlow lets users register, log in, create projects, and manage tasks — with status filters, assignee selection, optimistic UI updates, and a dark mode that persists across sessions.

| Layer | Choice |
|---|---|
| Backend | Go 1.22 + [chi](https://github.com/go-chi/chi) router |
| Database | PostgreSQL 15 |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) (embedded, auto-run on startup) |
| Auth | bcrypt (cost 12) + JWT (24 h expiry, stateless) |
| Frontend | React 18 + TypeScript + Vite |
| State | Zustand with `persist` middleware |
| Styling | Custom CSS — Zomato-inspired card UI, dark mode |
| Containers | Docker + Docker Compose |

---

## 2. Architecture Decisions

### `handler → store` — no service layer
Handlers own HTTP concerns (parsing, validation, status codes).
Stores own SQL (queries, scanning, error wrapping).
No service layer was added — the business logic is simple enough that one would add indirection without benefit. If domain logic grew significantly, a service layer would be the first thing I'd introduce.

### Migrations embedded in the binary
SQL files live in `backend/db/migrations/` and are embedded via `//go:embed` at compile time. The backend applies them automatically on startup using `golang-migrate`. `docker compose up` is truly zero-step — no separate migration command needed.

### Seed data as a migration
The test user and demo data are in `000002_seed.up.sql`. They run on first startup alongside the schema, making the reviewer experience friction-free: clone → compose up → log in.

### JWT is stateless
No token blacklist, no refresh tokens. Tokens expire after 24 hours. Sufficient for a take-home; a production system would add refresh tokens and a revocation store.

### 401 vs 403 are always distinct
`401 Unauthorized` — missing or invalid token (not authenticated).
`403 Forbidden` — valid token, wrong user (authenticated, but not allowed).
These are separate code paths throughout all handlers.

### Frontend state with Zustand
Auth state (token + user) is persisted to `localStorage` via Zustand's `persist` middleware. The app hydrates immediately on refresh without hitting the backend. Dark mode preference is persisted in the same way.

### Optimistic task status updates
Changing a task's status updates the UI immediately, then syncs to the API. If the API call fails, the previous state is restored and an error banner is shown — no silent failures.

### What was intentionally left out
- No service layer (handler → store is sufficient for this scope)
- No refresh tokens (out of scope)
- No drag-and-drop (not required)
- No pagination (not required; first addition at scale)
- `GET /users` returns all users — in production would be scoped to project members

---

## 3. Project Structure

```
taskflow-abhinav/
├── docker-compose.yml          # Starts postgres + backend + frontend
├── .env.example                # Environment variable template
│
├── backend/
│   ├── cmd/server/main.go      # Entrypoint — config, DB, router, graceful shutdown
│   ├── config/config.go        # Loads DATABASE_URL, JWT_SECRET, API_PORT
│   ├── db/
│   │   ├── db.go               # pgxpool setup + runs embedded migrations on startup
│   │   └── migrations/
│   │       ├── 000001_init.{up,down}.sql
│   │       ├── 000002_seed.{up,down}.sql
│   │       └── 000003_project_updated_at.{up,down}.sql
│   ├── handler/
│   │   ├── auth.go             # POST /auth/register, POST /auth/login
│   │   ├── projects.go         # CRUD /projects
│   │   ├── tasks.go            # CRUD /projects/:id/tasks + /tasks/:id
│   │   ├── users.go            # GET /users
│   │   └── helpers.go          # encode, decode, writeError, writeValidationError
│   ├── middleware/
│   │   ├── auth.go             # JWT → injects user_id + email into context
│   │   └── logger.go           # Structured request log
│   ├── model/models.go         # User, Project, Task structs (json:"-" on password)
│   ├── store/
│   │   ├── user.go             # CreateUser, GetUserByEmail, ListUsers
│   │   ├── project.go          # Full project CRUD
│   │   └── task.go             # Full task CRUD + dynamic filters
│   ├── docs/openapi.json       # OpenAPI 3.0 spec → served at /docs/
│   ├── seeds/                  # Manual re-seed + cleanup helpers
│   ├── Dockerfile              # Multi-stage build (builder → slim runtime)
│   └── Makefile                # tidy | test | build | run | dev
│
└── frontend/
    ├── src/
    │   ├── main.tsx             # Entry — applies stored theme before React hydrates
    │   ├── App.tsx              # Router + syncs html.dark class from theme store
    │   ├── styles.css           # Full design system + dark mode overrides
    │   ├── api/client.ts        # Axios + authApi, projectsApi, tasksApi, usersApi
    │   ├── store/
    │   │   ├── auth.ts          # Zustand persisted — token + user
    │   │   └── theme.ts         # Zustand persisted — dark toggle
    │   ├── types/index.ts       # TypeScript interfaces for all API shapes
    │   ├── components/
    │   │   ├── Navbar.tsx           # Brand, user, dark mode toggle, logout
    │   │   ├── ProtectedRoute.tsx   # Redirects unauthenticated → /login
    │   │   ├── ProjectCard.tsx      # Project card with hover animation
    │   │   ├── CreateProjectModal.tsx
    │   │   ├── TaskCard.tsx         # Inline status change + assignee/date chips
    │   │   └── TaskModal.tsx        # Create/edit task (real user dropdown)
    │   └── pages/
    │       ├── LoginPage.tsx / RegisterPage.tsx
    │       ├── ProjectsPage.tsx     # Skeleton loading + empty state
    │       ├── ProjectDetailPage.tsx # Filters + optimistic updates
    │       └── NotFoundPage.tsx
    ├── Dockerfile               # Vite build → nginx
    └── vite.config.ts
```

---

## 4. Running Locally

###  Option A — Docker Compose (recommended)

> Requires: **Docker Desktop only**. Nothing else needs to be installed.

```bash
git clone https://github.com/abhinav2712/taskflow-abhinav.git
cd taskflow-abhinav
cp .env.example .env
docker compose up --build
```

| Service | URL |
|---|---|
|  Frontend | http://localhost:3000 |
|  Backend API | http://localhost:8080 |
|  API Docs (OpenAPI) | http://localhost:8080/docs/ |
|  PostgreSQL | localhost:5432 |

**What happens automatically:**
- Postgres starts with a healthcheck
- Backend waits for Postgres, then runs all migrations (schema + seed) on startup
- Frontend is served via nginx on port 3000

**Reset everything (wipe DB + reseed):**
```bash
docker compose down -v && docker compose up --build
```

---

###  Option B — Run services separately

> Requires: Go 1.22+, Node 18+, a running PostgreSQL instance.

**Backend:**
```bash
cp .env.example .env
# Edit .env — set DATABASE_URL to your local Postgres

cd backend
go mod tidy
make run        # auto-applies migrations on startup
# make dev      # hot reload via air (go install github.com/air-verse/air@latest)
```

**Frontend:**
```bash
echo "VITE_API_URL=http://localhost:8080" > frontend/.env
cd frontend
npm install
npm run dev     # http://localhost:5173
```

> CORS defaults to `http://localhost:3000` (Docker) and `http://localhost:5173` (Vite dev).
> For Railway or any hosted frontend, set `ALLOWED_ORIGINS` to your deployed frontend URL.
> The backend also supports Railway's injected `PORT` automatically, and you can still set `API_PORT` explicitly for local or other environments.

---

## 5. Running Migrations

Migrations run **automatically** on every backend startup — no manual step required.

```
backend/db/migrations/
├── 000001_init.{up,down}.sql                  ← users, projects, tasks schema
├── 000002_seed.{up,down}.sql                  ← test user + demo project + tasks
└── 000003_project_updated_at.{up,down}.sql    ← added updated_at to projects
```

They are embedded in the Go binary (`//go:embed`) and applied via `golang-migrate`. State is tracked in `schema_migrations`.

**Manual seed helpers** (if schema already exists):
```bash
psql "$DATABASE_URL" -f backend/seeds/test_data.sql   # re-apply seed data
psql "$DATABASE_URL" -f backend/seeds/cleanup.sql      # remove seed data
```

---

## 6. Test Credentials

> The seed migration creates a test account automatically. No registration step needed.

```
Email:    test@example.com
Password: password123
```

The seed also creates:
-  1 demo project: **"Website Redesign"**
-  3 demo tasks with statuses: `todo`, `in_progress`, `done`

---

## 7. What I'd Do With More Time

**Features**
- **Drag-and-drop** — move tasks between status groups (`todo → in_progress → done`) with optimistic updates and API rollback on failure
- **Scoped user list** — `GET /users` currently returns all users system-wide; would scope it to members of the project (owners + assignees) in production
- **Null-clearing in PATCH** — assignee and due date can be set but not cleared back to `NULL`; would introduce a three-state nullable type (`present / null / value`) for those fields

**Frontend**
- **Lazy-load project detail page** — split the bundle so `/projects/:id` loads separately, reducing initial load time
- **Unit tests** — Jest + React Testing Library for auth forms, optimistic update logic, and task modal
- **Toast notifications** — replace inline feedback banners with proper toasts for creates/updates/deletes

**Backend**
- **Integration tests** — full handler-level HTTP tests for all routes (a `tests/` skeleton already exists)
- **Rate limiting** — prevent abuse on auth routes (`/auth/register`, `/auth/login`)
- **Redis caching** — cache frequently read data (project lists, user lists) to reduce DB load
- **Webhooks** — emit events on task status changes for external integrations
- **More granular error codes** — distinguish between different 400/500 failure sub-types instead of generic messages

**Infrastructure**
- **Backend healthcheck** — add an explicit `GET /healthz` endpoint and wire it into `docker-compose.yml`
- **Smaller production images** — switch to distroless or minimal Alpine base for both backend and frontend Docker images
- **CI/CD pipelines** — automate `go test`, `npm run build`, and linting on every pull request via GitHub Actions

---

## 9. AI Assistance Disclosure

This project was developed with AI assistance, with manual oversight, testing, and all architectural decisions made by me.

| Phase | Tool | Role |
|---|---|---|
| Planning & architecture | Claude | Draft architecture, folder structure, execution plan |
| Backend implementation | Codex | Auth, projects, tasks APIs per the planned architecture |
| Frontend implementation | Codex | React app scaffold, auth flow, routing, projects/tasks UI |
| Review & refinement | Manual | Auth edge cases, 401/403 separation, optimistic rollback, UX |

All AI-generated code was reviewed, tested manually (curl + browser), and adjusted to ensure correctness and alignment with the assignment requirements.
