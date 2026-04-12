# TaskFlow — Architecture Plan

> Full Stack Engineer · Go + React · 4–5 hour budget

---

## 1. High-Level Architecture

```
Browser (React SPA)
      │  REST/JSON
      ▼
Go API (chi router) ──► PostgreSQL 15
      │
      └─ golang-migrate runs on startup
```

- Single Go binary. No service mesh, no message bus, no GraphQL.
- React SPA served by its own Vite dev server (dev) or Nginx container (compose prod).
- JWT is stateless — no token store needed.
- All env config via a single `.env` file at the repo root.

---

## 2. Repo Structure

```
taskflow-abhinav/
├── docker-compose.yml
├── .env.example
├── README.md
│
├── backend/
│   ├── Dockerfile                  # multi-stage
│   ├── go.mod / go.sum
│   ├── cmd/
│   │   └── server/
│   │       └── main.go             # wires everything, graceful shutdown
│   ├── migrations/                 # top-level — easy to mount, easy to find
│   │   ├── 000001_init.up.sql
│   │   ├── 000001_init.down.sql
│   │   ├── 000002_seed.up.sql
│   │   └── 000002_seed.down.sql
│   ├── config/
│   │   └── config.go               # reads env vars
│   ├── db/
│   │   └── db.go                   # pgx pool init + RunMigrations
│   ├── middleware/
│   │   ├── auth.go                 # JWT validation, injects user into ctx
│   │   └── logger.go               # slog request logging
│   ├── handler/
│   │   ├── auth.go
│   │   ├── projects.go
│   │   └── tasks.go
│   ├── store/
│   │   ├── user.go
│   │   ├── project.go
│   │   └── task.go
│   └── model/
│       └── models.go               # plain structs (User, Project, Task)
│
└── frontend/
    ├── Dockerfile                  # Nginx, serves built dist/
    ├── index.html
    ├── vite.config.ts
    ├── tsconfig.json
    ├── package.json
    └── src/
        ├── main.tsx
        ├── App.tsx                 # Router setup
        ├── api/
        │   └── client.ts           # axios instance + interceptors
        ├── store/
        │   └── auth.ts             # Zustand auth slice
        ├── pages/
        │   ├── LoginPage.tsx
        │   ├── RegisterPage.tsx
        │   ├── ProjectsPage.tsx
        │   ├── ProjectDetailPage.tsx
        │   └── NotFoundPage.tsx
        ├── components/
        │   ├── Navbar.tsx
        │   ├── ProtectedRoute.tsx
        │   ├── TaskCard.tsx
        │   ├── TaskModal.tsx       # create / edit
        │   ├── ProjectCard.tsx
        │   └── CreateProjectModal.tsx
        └── types/
            └── index.ts            # TypeScript interfaces
```

> **Why flat handler/store dirs?** Two layers (handler → store) is all you need for this scope. Adding a "service" layer would be ceremony, not value.

---

## 3. Backend Modules

### `config`
Reads env vars with sensible defaults. Panics on missing JWT_SECRET (fails fast in dev/CI).

### `db`
- `pgxpool` for connection pooling.
- `golang-migrate` runs automatically **before** the HTTP server starts — no sidecar, no manual step.
- Migrations source: `//go:embed` points at `backend/migrations/*.sql` (top-level, not nested under `db/`) — easier to reason about and easier to mount as a Docker volume if needed.

### `middleware`
- **`auth.go`**: Parses `Authorization: Bearer <token>`, validates JWT, injects `userID` + `email` into `context`. Returns `401` if missing/invalid.
- **`logger.go`**: `slog` structured request logs (method, path, status, latency).

### `store` (data layer)
Plain functions that take a `*pgxpool.Pool` and return typed results or errors. No ORM, no repository interface — injected via closure at startup.

```
store/user.go    → CreateUser, GetByEmail
store/project.go → Create, List (owner OR assignee), Get, Update, Delete
store/task.go    → List (with filters), Create, Get, Update, Delete, GetStats
```

### `handler` (HTTP layer)
Chi handlers that: decode request body → validate → call store → encode response. Each handler file maps 1:1 to a resource. Error helper centralises status code → JSON mapping.

### `cmd/server/main.go`
Lives at `backend/cmd/server/main.go` — standard Go project layout. The `cmd/` convention signals "this is an entrypoint binary", not a library package. Built with `go build ./cmd/server`.

1. Load config
2. Init DB pool
3. Run migrations
4. Build and mount router  
5. `http.ListenAndServe` with `context` tied to `SIGTERM`/`SIGINT` for graceful shutdown

---

## 4. Frontend Modules

### `api/client.ts`
Axios instance with:
- `baseURL` from `VITE_API_URL`
- Request interceptor: injects `Authorization: Bearer <token>` from Zustand store
- Response interceptor: on `401` → clear store → redirect `/login`

### `store/auth.ts` (Zustand)
```ts
{ token, user, setAuth, clearAuth }
```
Persisted to `localStorage` via `zustand/middleware`. This is the single source of auth truth — no React context needed.

### Pages
| Page | Key behaviour |
|---|---|
| `LoginPage` / `RegisterPage` | React Hook Form + zod validation, maps API errors to field-level messages |
| `ProjectsPage` | Lists projects, "New Project" opens `CreateProjectModal` |
| `ProjectDetailPage` | Fetches project + tasks, filter bar (status / assignee), task list with `TaskCard` |
| `TaskModal` | Create or edit — same form. Optimistic update for status change. |

### `ProtectedRoute`
Wrapper that checks Zustand for token. If absent → `<Navigate to="/login" />`.

### State management
- Auth: Zustand (persisted)
- Server data: plain `useState` + `useEffect` with explicit loading/error flags — **no React Query**. Simple is fast to write and easy to explain.

> [!NOTE]
> React Query would be more production-correct for caching + refetch, but it adds setup time and complexity for a 4-hour exercise. `useState` + `useEffect` with proper loading/error states is completely defensible and reviewable.

---

## 5. Database Schema

```sql
-- 001_init.up.sql

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    email       TEXT NOT NULL UNIQUE,
    password    TEXT NOT NULL,              -- bcrypt hash
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    description TEXT,
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'todo'
                    CHECK (status IN ('todo', 'in_progress', 'done')),
    priority    TEXT NOT NULL DEFAULT 'medium'
                    CHECK (priority IN ('low', 'medium', 'high')),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    creator_id  UUID NOT NULL REFERENCES users(id),   -- for delete auth
    due_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- indexes for common query patterns
CREATE INDEX idx_tasks_project_id   ON tasks(project_id);
CREATE INDEX idx_tasks_assignee_id  ON tasks(assignee_id);
CREATE INDEX idx_tasks_status       ON tasks(project_id, status);
CREATE INDEX idx_projects_owner_id  ON projects(owner_id);
```

**Answers the open question:** Yes, add `creator_id` to tasks. The assignment says "project owner **or task creator**" can delete — you need it.

### seed.sql
```sql
-- seed.sql (run once after migrations)
INSERT INTO users (id, name, email, password) VALUES
  ('00000000-0000-0000-0000-000000000001',
   'Test User', 'test@example.com',
   '$2a$12$...bcrypt hash of "password123"...');

INSERT INTO projects (id, name, description, owner_id) VALUES
  ('00000000-0000-0000-0000-000000000010',
   'Demo Project', 'Seeded project for reviewers',
   '00000000-0000-0000-0000-000000000001');

INSERT INTO tasks (title, status, priority, project_id, creator_id) VALUES
  ('First task',   'todo',        'high',   '...project_id...', '...user_id...'),
  ('Second task',  'in_progress', 'medium', '...project_id...', '...user_id...'),
  ('Third task',   'done',        'low',    '...project_id...', '...user_id...');
```

Pre-compute the bcrypt hash in Go: `go run ./scripts/hashpw/main.go password123`.

---

## 6. API Design Decisions

### Auth returns user object alongside token
```json
// POST /auth/login → 200
{ "token": "<jwt>", "user": { "id": "...", "name": "...", "email": "..." } }
```
Frontend needs user name for Navbar — avoids an extra `/me` round-trip.

### Project list logic
`GET /projects` returns projects where `owner_id = me` **OR** `assignee_id = me` (via JOIN on tasks). Single query:
```sql
SELECT DISTINCT p.* FROM projects p
LEFT JOIN tasks t ON t.project_id = p.id
WHERE p.owner_id = $1 OR t.assignee_id = $1
ORDER BY p.created_at DESC;
```

### Project detail includes tasks inline
`GET /projects/:id` returns project + tasks array. One endpoint, one page-load, no waterfall.

### Task filtering
`GET /projects/:id/tasks?status=todo&assignee=uuid` — filters composed in SQL with optional `WHERE` clauses. No separate endpoint needed.

### PATCH semantics
Only include fields that are changing — the handler uses pointer fields in the request struct (`*string`, `*uuid.UUID`). The store builds the SET clause from non-nil fields only. PATCH, not PUT.

### `updated_at` handling
`updated_at` is **not** managed by a DB trigger. It is set explicitly in every UPDATE query:
```sql
UPDATE tasks
SET title = COALESCE($1, title),
    status = COALESCE($2, status),
    ...,
    updated_at = now()
WHERE id = $n
RETURNING *;
```
This is intentional — triggers are invisible magic that makes queries harder to reason about. Explicit `updated_at = now()` in the UPDATE statement is always correct, always visible, and reviewers can confirm it at a glance.

### HTTP status codes (strict)
| Scenario | Code |
|---|---|
| Created | 201 |
| Updated / fetched | 200 |
| Deleted | 204 |
| Validation failure | 400 |
| Missing / invalid token | 401 |
| Valid token, wrong owner | 403 |
| Resource not found | 404 |

### Error response shape
```json
{ "error": "validation failed", "fields": { "title": "is required" } }
{ "error": "not found" }
{ "error": "forbidden" }
```

---

## 7. Docker / Runtime Design

```yaml
# docker-compose.yml (simplified)
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB:       ${POSTGRES_DB}
      POSTGRES_USER:     ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 5s
      retries: 5

  api:
    build: ./backend
    env_file: .env
    ports: ["8080:8080"]
    depends_on:
      postgres:
        condition: service_healthy

  frontend:
    build: ./frontend
    ports: ["3000:80"]
    depends_on: [api]
```

### Backend Dockerfile (multi-stage)
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /taskflow-api ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /taskflow-api /taskflow-api
ENTRYPOINT ["/taskflow-api"]
```

### Frontend Dockerfile
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
```

> `nginx.conf` must include `try_files $uri /index.html;` for React Router to work.

### `.env.example`
```
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=taskflow
POSTGRES_USER=taskflow
POSTGRES_PASSWORD=taskflow_secret
DATABASE_URL=postgres://taskflow:taskflow_secret@postgres:5432/taskflow?sslmode=disable
JWT_SECRET=change_me_in_production_32_chars_min
API_PORT=8080
VITE_API_URL=http://localhost:8080
```

---

## 8. Implementation Order

Work in this sequence to always have something runnable:

| Step | What | Time est. |
|---|---|---|
| 1 | Scaffold: `go mod init`, Vite init, docker-compose skeleton | 15 min |
| 2 | DB schema + migration files + seed.sql | 20 min |
| 3 | Backend: config, db pool, migrations auto-run | 20 min |
| 4 | Backend: models, user store, auth handlers (register/login) | 30 min |
| 5 | Backend: JWT middleware | 15 min |
| 6 | Backend: project store + handlers | 30 min |
| 7 | Backend: task store + handlers (incl. filters) | 30 min |
| 8 | Smoke-test API with curl / Bruno | 10 min |
| 9 | Frontend: Zustand auth store + axios client | 20 min |
| 10 | Frontend: Login + Register pages | 30 min |
| 11 | Frontend: Protected route + Navbar | 15 min |
| 12 | Frontend: ProjectsPage + CreateProjectModal | 30 min |
| 13 | Frontend: ProjectDetailPage + filter bar | 30 min |
| 14 | Frontend: TaskModal (create + edit) | 30 min |
| 15 | Docker compose end-to-end test | 15 min |
| 16 | README | 20 min |
| 17 | **Bonus**: stats endpoint, 3 integration tests, dark mode | 30 min |

**Total: ~5 hours.** Steps 1–8 unblock 9–14. Don't start the frontend until auth endpoints pass curl.

---

## 9. Risks and How to Avoid Them

| Risk | Mitigation |
|---|---|
| Docker: api starts before postgres is ready | `depends_on: condition: service_healthy` on postgres healthcheck |
| Migrations fail silently | `log.Fatal` if migrate returns non-nil, non-`ErrNoChange` error |
| JWT secret in source | Read from env — panic loudly if `JWT_SECRET` is empty |
| CORS issues in compose | Add chi CORS middleware with `AllowedOrigins: ["http://localhost:3000"]` |
| React Router 404 on refresh | Nginx `try_files $uri /index.html` |
| `task.creator_id` missing → wrong delete auth | Include it in schema from day 1, populate on create |
| `PATCH` overwriting fields with zero values | Use pointer fields in request struct; only `UPDATE` non-nil fields |
| Optimistic UI race condition | Keep a `previousState` before update, restore in the catch block |
| Time crunch | Do NOT start bonus until steps 1–16 are green |

---

## 10. What to Intentionally Skip

| Skipped | Reason |
|---|---|
| Role-based access control (RBAC) | Assignment only requires owner/assignee logic — done inline in handlers |
| Refresh tokens | 24h JWT is sufficient; no secure storage complexity needed |
| Rate limiting | Not in scope; mention in "What You'd Do With More Time" |
| Pagination | Implement only if steps 1–16 finish early (it's a bonus) |
| WebSocket / SSE | Same — bonus only |
| React Query / TanStack Query | `useState` + `useEffect` is simpler to read and explain in 30 min code review |
| Repository pattern / interfaces | No second implementation exists, so an interface is untestable theatre |
| Unit tests on handlers | Integration tests (bonus) are higher signal; pure unit tests on CRUD handlers are low value |
| `updated_at` trigger | Set `updated_at = now()` explicitly in every UPDATE SQL statement — explicit is better than a trigger that's invisible during code review |
| Separate migrator container | Run migrations in the API entrypoint — fewer moving parts |

---

## Open Questions (Resolved)

| Question | Decision |
|---|---|
| Add `task.creator_id`? | **Yes** — needed for delete authorization per assignment spec |
| Auto-run migrations or separate step? | **Auto-run in API entrypoint** before server starts — zero manual steps required |
| Grouped tasks UI or flat table? | **Flat list with filter bar** — simpler, fully responsive, faster to build |
| Minimal API response shapes? | Return full object on create/update; use consistent envelope `{ "projects": [] }` for lists, bare object for single resources |

## Authorization Rules

- Only authenticated users can access API
- Project owner can:
  - update project
  - delete project
  - create tasks
- Task creator OR project owner can:
  - delete task
- Assignee can:
  - update task status