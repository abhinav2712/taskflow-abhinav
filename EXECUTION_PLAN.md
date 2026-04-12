# TaskFlow — Execution Plan

> Phases in strict build order. Do not skip ahead. Each phase ends with a gate check.

---

## Phase 0 — Repo Setup

**Goal:** Working repo skeleton with all tooling in place. Nothing functional yet, but everything scaffolded so later phases are just filling in files.

### Files to Create
```
taskflow-abhinav/
├── .env.example
├── .gitignore               (extend existing)
├── docker-compose.yml       (skeleton — services stubbed)
├── backend/
│   ├── go.mod
│   └── main.go              (stub — just logs "starting")
└── frontend/                (Vite scaffold)
    ├── package.json
    ├── vite.config.ts
    ├── tsconfig.json
    └── src/main.tsx
```

### What to Implement
- [ ] Add `/backend`, `/frontend` to `.gitignore`'s vendor/node_modules patterns
- [ ] Write `.env.example` with all vars: `POSTGRES_*`, `DATABASE_URL`, `JWT_SECRET`, `API_PORT`, `VITE_API_URL`
- [ ] `docker-compose.yml` skeleton — postgres + api + frontend services, ports only, no env_file yet
- [ ] `cd backend && go mod init github.com/yourname/taskflow`
- [ ] Stub `main.go` that just prints "TaskFlow API starting" and exits
- [ ] `cd frontend && npm create vite@latest . -- --template react-ts`
- [ ] Install frontend deps: `npm install axios zustand react-router-dom react-hook-form zod @hookform/resolvers`
- [ ] Install shadcn/ui: follow `npx shadcn@latest init` — pick **Default** style, **Zinc** base color, CSS variables yes
- [ ] Add shadcn components needed upfront: `npx shadcn@latest add button input label dialog select badge`

### Gate Check ✅
- [ ] `go build ./...` in `/backend` succeeds (even with stub main)
- [ ] `npm run dev` in `/frontend` opens Vite default page at localhost:5173
- [ ] `.env.example` committed, `.env` in `.gitignore`

### Common Mistakes to Avoid
- Do **not** commit `.env` — `cp .env.example .env` is a manual step for the reviewer
- Do **not** run `npx shadcn` before installing base deps — it will fail
- Set `"baseUrl": "src"` in `tsconfig.json` now so import paths are clean throughout

---

## Phase 1 — Database and Migrations

**Goal:** Schema fully defined in SQL migration files. Postgres boots and migrations run cleanly.

### Files to Create
```
backend/
├── db/
│   ├── db.go
│   └── migrations/
│       ├── 000001_init.up.sql
│       ├── 000001_init.down.sql
│       └── seed.sql
└── config/
    └── config.go
```

### What to Implement

#### `config/config.go`
- Read `DATABASE_URL`, `JWT_SECRET`, `API_PORT` from env
- `log.Fatal` if `JWT_SECRET` is empty
- Fall back to assembling `DATABASE_URL` from `POSTGRES_*` parts if not set directly

#### `db/db.go`
- `New(databaseURL string) (*pgxpool.Pool, error)` — connect and ping
- `RunMigrations(databaseURL string)` — use `golang-migrate` with `iofs` source pointing at embedded `migrations/` dir
  ```go
  //go:embed migrations/*.sql
  var migrationsFS embed.FS
  ```
- `log.Fatal` on any migrate error except `migrate.ErrNoChange`

#### `000001_init.up.sql`
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL,
  email      TEXT NOT NULL UNIQUE,
  password   TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
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
  creator_id  UUID NOT NULL REFERENCES users(id),
  due_date    DATE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tasks_project_id  ON tasks(project_id);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX idx_tasks_status      ON tasks(project_id, status);
CREATE INDEX idx_projects_owner_id ON projects(owner_id);
```

#### `000001_init.down.sql`
```sql
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "pgcrypto";
```

#### `seed.sql`
- 1 user: `test@example.com` / `password123` (pre-hashed bcrypt cost 12)
- 1 project owned by that user
- 3 tasks with statuses: `todo`, `in_progress`, `done`
- Use fixed UUIDs so seed is idempotent with `ON CONFLICT DO NOTHING`

Pre-generate the bcrypt hash: `go run ./scripts/hashpw/main.go password123`

#### `main.go` (update stub)
- Load config → init DB pool → run migrations → `log.Info("migrations done")`

### Gate Check ✅
- [ ] `go run ./main.go` connects to a local postgres and logs "migrations done" without error
- [ ] `psql $DATABASE_URL -c '\dt'` shows `users`, `projects`, `tasks`
- [ ] Running migrations twice does not error (ErrNoChange is swallowed)
- [ ] Down migration drops all three tables cleanly

### Common Mistakes to Avoid
- golang-migrate needs the migration files named exactly `NNNNNN_name.up.sql` / `NNNNNN_name.down.sql` with leading zeros
- Embed directive must be in the **same package** as the `//go:embed` declaration
- Do NOT use postgres enums (`CREATE TYPE`) — `CHECK` constraints are simpler to extend
- Seed with `ON CONFLICT DO NOTHING` — running seed twice must be safe

---

## Phase 2 — Auth Backend

**Goal:** `POST /auth/register` and `POST /auth/login` work correctly and return a JWT.

### Files to Create
```
backend/
├── model/models.go
├── store/user.go
├── handler/
│   ├── helpers.go       (shared encode/decode/error helpers)
│   └── auth.go
└── middleware/
    ├── auth.go
    └── logger.go
```

### What to Implement

#### `model/models.go`
```go
type User    struct { ID uuid.UUID; Name, Email, Password string; CreatedAt time.Time }
type Project struct { ID uuid.UUID; Name string; Description *string; OwnerID uuid.UUID; CreatedAt time.Time; Tasks []Task }
type Task    struct { ID uuid.UUID; Title string; Description *string; Status, Priority string;
                      ProjectID uuid.UUID; AssigneeID *uuid.UUID; CreatorID uuid.UUID;
                      DueDate *string; CreatedAt, UpdatedAt time.Time }
```
- `Password` has `json:"-"` — never serialise

#### `store/user.go`
- `CreateUser(ctx, pool, name, email, hashedPassword) (User, error)`
- `GetUserByEmail(ctx, pool, email) (User, error)` — return `pgx.ErrNoRows` passthrough

#### `handler/helpers.go`
- `decode(r, &v) error` — JSON decode + return 400 on bad body
- `encode(w, status, v)` — JSON encode with correct Content-Type
- `writeError(w, status, msg string)` — `{"error": msg}`
- `writeValidationError(w, fields map[string]string)` — `{"error":"validation failed","fields":{...}}`

#### `handler/auth.go`
- **Register**: validate name/email/password present → check email not taken (409 if duplicate) → `bcrypt.GenerateFromPassword(cost=12)` → store → return `201 { token, user }`
- **Login**: look up by email → `bcrypt.CompareHashAndPassword` → generate JWT (claims: `user_id`, `email`, exp 24h) → return `200 { token, user }`
- JWT signed with `HS256`, secret from config

#### `middleware/auth.go`
- Extract `Authorization: Bearer <token>` header → `401` if missing
- Parse and validate JWT → `401` if invalid/expired
- Inject `userID` (uuid) and `email` (string) into `context`
- Export `UserIDFromCtx(ctx) uuid.UUID` helper

#### `middleware/logger.go`
- Wrap handler, capture status code via `responseWriter` wrapper
- `slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", status, "latency", duration)`

#### `main.go` (update)
- Mount CORS middleware (`go-chi/cors`) allowing `http://localhost:3000`
- Mount logger middleware
- `POST /auth/register` and `POST /auth/login` — **no auth middleware on these**

### Gate Check ✅
```bash
# Register
curl -s -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Test","email":"a@b.com","password":"secret123"}' | jq .
# → 201 { token, user }

# Login
curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"secret123"}' | jq .
# → 200 { token, user }

# Wrong password
# → 401 { "error": "unauthorized" }

# Duplicate email
# → 409 { "error": "email already in use" }

# Missing field
# → 400 { "error": "validation failed", "fields": { "email": "is required" } }
```

### Common Mistakes to Avoid
- bcrypt cost MUST be ≥ 12 — anything lower is an automatic disqualifier
- Return `401` for wrong password, **not** `400` or `404`
- Don't return `password` hash in any response — `json:"-"` on the field
- JWT secret must come from env — hardcoding it is an auto-disqualifier
- Duplicate email should be `409 Conflict`, not `400` (more semantically correct)

---

## Phase 3 — Projects Backend

**Goal:** All 5 project endpoints work with correct auth, ownership checks, and cascade delete.

### Files to Create
```
backend/
├── store/project.go
└── handler/projects.go
```

### What to Implement

#### `store/project.go`
- `CreateProject(ctx, pool, name string, desc *string, ownerID uuid.UUID) (Project, error)`
- `ListProjects(ctx, pool, userID uuid.UUID) ([]Project, error)`
  ```sql
  SELECT DISTINCT p.* FROM projects p
  LEFT JOIN tasks t ON t.project_id = p.id
  WHERE p.owner_id = $1 OR t.assignee_id = $1
  ORDER BY p.created_at DESC
  ```
- `GetProject(ctx, pool, id uuid.UUID) (Project, error)` — includes tasks in a second query
- `UpdateProject(ctx, pool, id uuid.UUID, name *string, desc *string) (Project, error)` — only update non-nil fields
- `DeleteProject(ctx, pool, id uuid.UUID) error` — cascade handles tasks

#### `handler/projects.go`
- `GET /projects` — list, return `{"projects": [...]}`
- `POST /projects` — validate name present → create → `201` with project
- `GET /projects/:id` — fetch → `404` if not found → return project with tasks array
- `PATCH /projects/:id` — check ownership (owner_id == caller) → `403` if not → update → `200`
- `DELETE /projects/:id` — check ownership → `403` if not → delete → `204`

#### `main.go` (update)
- Mount project routes under `/projects` with `auth` middleware applied to all

### Gate Check ✅
```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJleHAiOjE3NzYwODQ0MzMsImlhdCI6MTc3NTk5ODAzMywidXNlcl9pZCI6IjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMSJ9.n0DZUzBX3kRVEeEltyyvnUa-xTEPaB5XKg_uK5zrAo0"

# Create project
curl -s -X POST http://localhost:8080/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"My Project"}' | jq .
# → 201 { id, name, owner_id, ... }

# List projects
curl -s http://localhost:8080/projects -H "Authorization: Bearer $TOKEN" | jq .
# → { "projects": [...] }

# Get project (includes tasks array, empty for now)
curl -s http://localhost:8080/projects/<id> -H "Authorization: Bearer $TOKEN" | jq .

# Update — non-owner should get 403
# Delete — non-owner should get 403

# No token → 401
curl -s http://localhost:8080/projects
# → { "error": "unauthorized" }
```

### Common Mistakes to Avoid
- `GET /projects` must return **both** owned projects AND projects where user is assignee — don't just filter by `owner_id`
- `403` for wrong owner, `404` if project doesn't exist at all — don't conflate
- `DELETE` returns `204 No Content` — no body, no JSON
- `PATCH` with no body fields should be a no-op, not an error

---

## Phase 4 — Tasks Backend

**Goal:** All 4 task endpoints work, including status/assignee filters and correct delete authorization.

### Files to Create
```
backend/
├── store/task.go
└── handler/tasks.go
```

### What to Implement

#### `store/task.go`
- `ListTasks(ctx, pool, projectID uuid.UUID, status *string, assigneeID *uuid.UUID) ([]Task, error)`
  - Build WHERE clause dynamically: always filter by `project_id`, optionally add `AND status = $n`, `AND assignee_id = $n`
- `CreateTask(ctx, pool, projectID, creatorID uuid.UUID, input CreateTaskInput) (Task, error)`
- `GetTask(ctx, pool, id uuid.UUID) (Task, error)`
- `UpdateTask(ctx, pool, id uuid.UUID, input UpdateTaskInput) (Task, error)`
  - `UpdateTaskInput` uses pointer fields: `Title *string`, `Status *string`, `Priority *string`, `AssigneeID *uuid.UUID`, `DueDate *string`
  - Always set `updated_at = now()` on update
- `DeleteTask(ctx, pool, id uuid.UUID) error`

#### `handler/tasks.go`
- `GET /projects/:id/tasks` — parse `?status=` and `?assignee=` query params → call ListTasks → `{"tasks": [...]}`
- `POST /projects/:id/tasks` — verify project exists → create with `creator_id = callerID` → `201`
- `PATCH /tasks/:id` — fetch task → update → `200`
- `DELETE /tasks/:id` — fetch task → check: `task.CreatorID == callerID OR project.OwnerID == callerID` → `403` if neither → delete → `204`

#### Bonus (if time allows)
- `GET /projects/:id/stats` → `{"total": n, "by_status": {"todo": n, "in_progress": n, "done": n}}`

### Gate Check ✅
```bash
# Create task
curl -s -X POST http://localhost:8080/projects/<pid>/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"My Task","priority":"high","status":"todo"}' | jq .
# → 201 task object with creator_id set

# Filter by status
curl -s "http://localhost:8080/projects/<pid>/tasks?status=todo" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Filter by assignee
curl -s "http://localhost:8080/projects/<pid>/tasks?assignee=<uid>" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Update task (partial)
curl -s -X PATCH http://localhost:8080/tasks/<tid> \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"status":"done"}' | jq .
# → updated_at changed, status is "done", other fields unchanged

# Delete — non-creator non-owner → 403
# Delete — creator → 204
```

### Common Mistakes to Avoid
- PATCH must NOT overwrite `title` with empty string if `title` not in request body — use `*string` pointer fields
- `updated_at` must be updated on every PATCH — do it in SQL: `updated_at = now()`
- Delete auth: **both** conditions (creator OR project owner) must be checked, not just one
- Filters are optional — missing `?status=` means return all statuses
- `invalid UUID` in query param should return `400`, not a panic

---

## Phase 5 — Frontend Auth

**Goal:** Login and Register pages work end-to-end. Auth state persists on refresh. Protected routes redirect correctly.

### Files to Create
```
frontend/src/
├── types/index.ts
├── api/client.ts
├── store/auth.ts
├── App.tsx
├── components/ProtectedRoute.tsx
├── components/Navbar.tsx
├── pages/LoginPage.tsx
├── pages/RegisterPage.tsx
└── pages/NotFoundPage.tsx
```

### What to Implement

#### `types/index.ts`
```ts
export interface User { id: string; name: string; email: string; }
export interface Project { id: string; name: string; description?: string; owner_id: string; created_at: string; tasks?: Task[]; }
export interface Task { id: string; title: string; description?: string; status: 'todo' | 'in_progress' | 'done';
  priority: 'low' | 'medium' | 'high'; project_id: string; assignee_id?: string;
  creator_id: string; due_date?: string; created_at: string; updated_at: string; }
```

#### `store/auth.ts`
```ts
// Zustand store persisted to localStorage
{ token: string | null; user: User | null; setAuth(token, user): void; clearAuth(): void; }
```
Use `zustand/middleware`'s `persist` with `localStorage`.

#### `api/client.ts`
- Axios instance, `baseURL: import.meta.env.VITE_API_URL`
- Request interceptor: reads token from Zustand store, adds `Authorization: Bearer <token>`
- Response interceptor: if `401` → call `clearAuth()` → `window.location.href = '/login'`
- Export typed functions: `authApi.login(email, password)`, `authApi.register(name, email, password)`

#### `App.tsx`
```tsx
<BrowserRouter>
  <Routes>
    <Route path="/login"    element={<LoginPage />} />
    <Route path="/register" element={<RegisterPage />} />
    <Route element={<ProtectedRoute />}>
      <Route path="/"           element={<ProjectsPage />} />
      <Route path="/projects/:id" element={<ProjectDetailPage />} />
    </Route>
    <Route path="*" element={<NotFoundPage />} />
  </Routes>
</BrowserRouter>
```

#### `ProtectedRoute.tsx`
- Read token from Zustand — if null, `<Navigate to="/login" replace />`
- Otherwise `<Outlet />`

#### `Navbar.tsx`
- Show `user.name` from Zustand
- Logout button → `clearAuth()` → navigate to `/login`

#### `LoginPage.tsx` / `RegisterPage.tsx`
- React Hook Form + zod schema validation
- On submit: call API → `setAuth(token, user)` → navigate to `/`
- Show field-level errors from both zod and API response
- Show loading spinner on submit button
- Link between login ↔ register

### Gate Check ✅
- [ ] Register with new email → redirected to projects page, name shows in navbar
- [ ] Refresh page → still logged in (Zustand persisted)
- [ ] Navigate to `/` without token → redirected to `/login`
- [ ] Login with wrong password → error message shown (not a blank screen)
- [ ] Logout → redirected to `/login`, refresh stays on `/login`
- [ ] No console errors in browser devtools

### Common Mistakes to Avoid
- Zustand `persist` serialises to localStorage — make sure initial state is read correctly on mount
- Axios interceptor reads from Zustand **at request time** (inside the function), not at axios instance creation time
- Response interceptor: only redirect on `401`, not all errors
- `ProtectedRoute` must use `<Outlet />` with React Router v6, not `children` pattern
- Always show an error state — never leave the user on a blank spinner

---

## Phase 6 — Frontend Projects/Tasks UI

**Goal:** Full CRUD for projects and tasks. Filters work. Optimistic status update works.

### Files to Create
```
frontend/src/
├── pages/ProjectsPage.tsx
├── pages/ProjectDetailPage.tsx
├── components/ProjectCard.tsx
├── components/CreateProjectModal.tsx
├── components/TaskCard.tsx
└── components/TaskModal.tsx
```

### What to Implement

#### `api/client.ts` (extend)
Add typed functions:
```ts
projectsApi.list()
projectsApi.create(name, description?)
projectsApi.get(id)
projectsApi.update(id, data)
projectsApi.delete(id)

tasksApi.list(projectId, filters?)
tasksApi.create(projectId, data)
tasksApi.update(id, data)
tasksApi.delete(id)
```

#### `ProjectsPage.tsx`
- `useEffect` fetches `GET /projects` on mount
- Show loading skeleton / error state
- Render `<ProjectCard>` grid — empty state: "No projects yet. Create one to get started."
- "New Project" button opens `<CreateProjectModal>`
- On project create: push to local state (no full refetch needed)

#### `ProjectCard.tsx`
- Shows name, description, `created_at`
- Click → navigate to `/projects/:id`
- Delete button (owner only — compare `project.owner_id === user.id`)

#### `CreateProjectModal.tsx`
- shadcn `<Dialog>` with name + description fields
- On submit: call API → close modal → add to list
- Loading state on submit button

#### `ProjectDetailPage.tsx`
- Fetch `GET /projects/:id` — shows project name + tasks
- Filter bar: status dropdown (All / Todo / In Progress / Done) + assignee input
- Re-fetch tasks when filters change: call `GET /projects/:id/tasks?status=...`
- "New Task" button opens `<TaskModal mode="create">`
- Render `<TaskCard>` list — empty state: "No tasks match this filter."

#### `TaskCard.tsx`
- Shows title, status badge (colour-coded), priority, due date, assignee
- Click → opens `<TaskModal mode="edit">`
- Status change via inline select — **optimistic update**:
  ```ts
  const prev = tasks
  setTasks(tasks.map(t => t.id === id ? {...t, status} : t))  // optimistic
  try { await tasksApi.update(id, { status }) }
  catch { setTasks(prev) }  // revert on error
  ```
- Delete button (creator or project owner only)

#### `TaskModal.tsx`
- Single component handles create + edit via `mode` prop
- Fields: title (required), description, status, priority, assignee (user ID text field for now), due_date
- On submit: call create or update API → close modal → update local list
- Show API validation errors inline

### Gate Check ✅
- [ ] Create project → appears in list immediately
- [ ] Click project → see detail page with task list
- [ ] Create task → appears in list, close modal
- [ ] Edit task → fields pre-populated, changes saved
- [ ] Filter by status → list updates
- [ ] Change task status inline → updates instantly (optimistic), reverts if API fails
- [ ] Delete task → removed from list
- [ ] Delete project → removed from list, cascades tasks on backend
- [ ] All loading states visible — no blank screens during fetch
- [ ] Empty states shown — no `undefined` rendered anywhere
- [ ] Works at 375px (mobile) — no horizontal overflow

### Common Mistakes to Avoid
- Always handle 3 states per async call: `loading`, `error`, `data` — never skip error state
- Empty state must be explicit: check `data.length === 0` not `!data`
- Optimistic update: save `prev` state **before** the async call, not inside catch
- Date input: use `type="date"` and format as `YYYY-MM-DD` when sending to API
- Don't leak stale closures in `useEffect` — include all dependencies in the deps array

---

## Phase 7 — Docker and Seed Data

**Goal:** `docker compose up` from a clean clone brings up the full stack with seed data. Zero manual steps.

### Files to Create/Update
```
backend/
├── Dockerfile
└── scripts/hashpw/main.go   (one-shot helper — generate bcrypt hash)
frontend/
├── Dockerfile
└── nginx.conf
docker-compose.yml            (finalise)
.env.example                  (verify all vars present)
backend/db/migrations/seed.sql (finalise with real bcrypt hash)
```

### What to Implement

#### `backend/Dockerfile`
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /taskflow-api .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /taskflow-api /taskflow-api
ENTRYPOINT ["/taskflow-api"]
```

#### `frontend/Dockerfile`
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL=http://localhost:8080
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

#### `frontend/nginx.conf`
```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

#### `docker-compose.yml` (final)
```yaml
version: "3.9"
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 5s
      timeout: 5s
      retries: 10

  api:
    build: ./backend
    env_file: .env
    ports: ["8080:8080"]
    depends_on:
      postgres:
        condition: service_healthy

  frontend:
    build:
      context: ./frontend
      args:
        VITE_API_URL: http://localhost:8080
    ports: ["3000:80"]
    depends_on: [api]

volumes:
  pgdata:
```

#### Seed data
- Run `go run ./scripts/hashpw/main.go password123` locally → paste hash into `seed.sql`
- Verify seed runs automatically: add a call to `db.RunSeed(pool)` in `main.go` after migrations (or embed seed in migration `000002`)
- Alternatively: add seed as a separate migration file `000002_seed.up.sql` — idempotent with `ON CONFLICT DO NOTHING`

### Gate Check ✅
```bash
# From fresh clone
cp .env.example .env
docker compose up --build

# Wait for all 3 services to be healthy, then:
curl http://localhost:8080/auth/login \
  -X POST -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"password123"}' | jq .
# → 200 { token, user }

# App at http://localhost:3000
# Login with test@example.com / password123
# See seed project with 3 tasks
```

- [ ] All 3 containers start without manual intervention
- [ ] Seed user can log in immediately
- [ ] API multi-stage build — final image is NOT `golang:1.22`
- [ ] Frontend refresh at `/projects/:id` does not 404

### Common Mistakes to Avoid
- `VITE_API_URL` is baked into the frontend at **build time** — pass it as build arg
- `depends_on: service_healthy` only works if the healthcheck is defined on the postgres service
- Seed must be idempotent (`ON CONFLICT DO NOTHING`) — running compose twice must not fail
- Nginx must serve on port 80, compose maps it to 3000: `"3000:80"`
- Copy `tzdata` into the alpine runtime image if your app touches time zones

---

## Phase 8 — README and Final Validation

**Goal:** README passes rubric. No console errors. No broken layouts. No disqualifiers.

### Files to Create/Update
```
README.md
```

### README Sections (all required by rubric)

1. **Overview** — what it is, what it does, tech stack list
2. **Architecture Decisions** — layered arch, why no ORM, why no React Query, PATCH semantics, what was intentionally skipped + why
3. **Running Locally**
   ```bash
   git clone https://github.com/yourname/taskflow-abhinav
   cd taskflow-abhinav
   cp .env.example .env
   docker compose up
   # Frontend: http://localhost:3000
   # API:      http://localhost:8080
   ```
4. **Running Migrations** — "Migrations run automatically on API container start via golang-migrate."
5. **Test Credentials**
   ```
   Email:    test@example.com
   Password: password123
   ```
6. **API Reference** — table of all endpoints with request/response examples (copy from architecture plan)
7. **What You'd Do With More Time** — honest list: rate limiting, refresh tokens, React Query, drag-and-drop, pagination, real WebSocket updates, proper test coverage

### Final Validation Checklist
- [ ] `docker compose up` from zero — no manual steps required
- [ ] Login with seed credentials → see seeded project with 3 tasks
- [ ] Register new user → create project → create tasks → edit tasks → delete tasks
- [ ] Filter tasks by status → correct subset shown
- [ ] Change task status inline → optimistic update visible
- [ ] Non-owner cannot delete another user's project (try with two accounts)
- [ ] Task creator (non-owner) can delete their own task
- [ ] Open DevTools — zero console errors in the production build
- [ ] Resize to 375px — no horizontal scrollbar, no broken layout
- [ ] Resize to 1280px — layout looks polished
- [ ] Refresh on `/projects/:id` → page loads (not 404)
- [ ] `.env` is NOT committed to git
- [ ] `JWT_SECRET` is NOT hardcoded anywhere in source
- [ ] `password` field never appears in any API response

### Common Mistakes to Avoid
- "What I'd do with more time" must be **honest**, not a list of features you secretly think aren't important — reviewers read this
- README must have exact commands, not "something like this"
- Double-check API reference matches what the code actually does — discrepancies look bad in code review
- Run `docker compose down -v && docker compose up --build` one final time before submitting

---

## Time Budget Summary

| Phase | Est. Time |
|---|---|
| 0 — Repo setup | 20 min |
| 1 — DB + migrations | 25 min |
| 2 — Auth backend | 35 min |
| 3 — Projects backend | 30 min |
| 4 — Tasks backend | 35 min |
| 5 — Frontend auth | 35 min |
| 6 — Frontend UI | 60 min |
| 7 — Docker + seed | 25 min |
| 8 — README + validation | 25 min |
| **Total** | **~5 hours** |

> Phases 0–4 must be done before touching the frontend. Phase 7 can be done in parallel with Phase 6 if comfortable with Docker.
