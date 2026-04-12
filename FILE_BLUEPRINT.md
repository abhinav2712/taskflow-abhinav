# TaskFlow — File-by-File Blueprint

> Implementation-ready reference. Every file in the monorepo, in dependency order.
> Do not write full code yet — use this as a contract before you open your editor.

---

## Conventions

- **Depends on** = other files in this repo this file imports
- **Do NOT put here** = things that belong in a different file (prevents scope creep)
- Backend package path root: `github.com/yourname/taskflow`
- Frontend src root: `src/`

---

# BACKEND

---

## `backend/go.mod`

**Purpose:** Go module definition and dependency pinning.

**Contents:**
```
module github.com/yourname/taskflow
go 1.22

require:
  github.com/go-chi/chi/v5
  github.com/go-chi/cors
  github.com/golang-jwt/jwt/v5
  github.com/golang-migrate/migrate/v4
  github.com/google/uuid
  github.com/jackc/pgx/v5
  golang.org/x/crypto
```

**Depends on:** nothing

**Do NOT put here:** indirect deps (go mod tidy handles those), replace directives unless necessary

---

## `backend/config/config.go`

**Purpose:** Single place to read and validate all environment variables. Called once at startup.

**Functions:**
```go
func Load() Config
func getEnv(key, fallback string) string  // unexported helper
```

**Types:**
```go
type Config struct {
    DatabaseURL string
    JWTSecret   string
    Port        string
}
```

**Depends on:** `os` (stdlib only)

**Do NOT put here:** DB connection logic, JWT signing, any business logic, constants for business rules

---

## `backend/model/models.go`

**Purpose:** Plain Go structs that represent domain entities. Shared across store and handler layers.

**Types:**
```go
type User struct {
    ID        uuid.UUID
    Name      string
    Email     string
    Password  string    // json:"-"
    CreatedAt time.Time
}

type Project struct {
    ID          uuid.UUID
    Name        string
    Description *string   // nullable
    OwnerID     uuid.UUID
    CreatedAt   time.Time
    Tasks       []Task    // json:"tasks,omitempty" — only populated on GET /projects/:id
}

type Task struct {
    ID          uuid.UUID
    Title       string
    Description *string
    Status      string     // "todo" | "in_progress" | "done"
    Priority    string     // "low" | "medium" | "high"
    ProjectID   uuid.UUID
    AssigneeID  *uuid.UUID // nullable
    CreatorID   uuid.UUID
    DueDate     *string    // "YYYY-MM-DD" or nil
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Depends on:** `github.com/google/uuid`, `time` (stdlib)

**Do NOT put here:** SQL queries, HTTP logic, validation logic, request/response types (those live in handler), methods on structs (keep it plain data)

---

## `backend/db/db.go`

**Purpose:** Initialize the pgx connection pool and run migrations. Called once in `main.go`.

**Functions:**
```go
func NewPool(databaseURL string) (*pgxpool.Pool, error)
// Connects to postgres, calls pool.Ping(), returns ready pool

func RunMigrations(databaseURL string)
// Uses golang-migrate with embedded FS source
// log.Fatal on any error except migrate.ErrNoChange
```

**Embed directive (in this file):**
```go
//go:embed migrations/*.sql
var migrationsFS embed.FS
```

**Depends on:** `config` (for databaseURL passed in), `jackc/pgx/v5`, `golang-migrate/migrate/v4`

**Do NOT put here:** SQL query functions (those live in `store/`), seed logic, business rules

---

## `backend/db/migrations/000001_init.up.sql`

**Purpose:** Create all tables and indexes. The single source of truth for the schema.

**Contains:**
- `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`
- `CREATE TABLE users` — id (uuid PK, gen_random_uuid()), name, email (unique), password, created_at
- `CREATE TABLE projects` — id, name, description (nullable), owner_id (FK users CASCADE), created_at
- `CREATE TABLE tasks` — id, title, description, status (CHECK constraint), priority (CHECK constraint), project_id (FK projects CASCADE), assignee_id (FK users SET NULL, nullable), creator_id (FK users), due_date (DATE nullable), created_at, updated_at
- 4 indexes: `idx_tasks_project_id`, `idx_tasks_assignee_id`, `idx_tasks_status(project_id, status)`, `idx_projects_owner_id`

**Depends on:** nothing

**Do NOT put here:** seed data, stored procedures, triggers, enum types (use CHECK constraints)

---

## `backend/db/migrations/000001_init.down.sql`

**Purpose:** Cleanly reverse the init migration.

**Contains:**
```sql
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "pgcrypto";
```

**Do NOT put here:** partial rollbacks — drop ALL tables created in the up file

---

## `backend/db/migrations/000002_seed.up.sql`

**Purpose:** Insert known test data. Idempotent — safe to run multiple times.

**Contains:**
- 1 user: `test@example.com`, bcrypt hash of `password123` (cost=12), fixed UUID
- 1 project owned by that user, fixed UUID
- 3 tasks: one each of `todo`, `in_progress`, `done`
- All INSERTs use `ON CONFLICT DO NOTHING`

**Depends on:** `000001_init.up.sql` must have run first

**Do NOT put here:** test-specific data beyond the 3 required items, UUIDs that collide with real production data patterns

---

## `backend/db/migrations/000002_seed.down.sql`

**Purpose:** Remove seed data.

**Contains:**
```sql
DELETE FROM tasks   WHERE project_id = '<seed-project-uuid>';
DELETE FROM projects WHERE id = '<seed-project-uuid>';
DELETE FROM users    WHERE id = '<seed-user-uuid>';
```

---

## `backend/middleware/auth.go`

**Purpose:** JWT validation middleware. Extracts user identity from the token and injects it into the request context.

**Functions:**
```go
func Authenticate(jwtSecret string) func(http.Handler) http.Handler
// Returns chi-compatible middleware
// 1. Read Authorization header
// 2. Strip "Bearer " prefix → 401 if missing
// 3. Parse JWT with jwtSecret → 401 if invalid or expired
// 4. Inject userID (uuid.UUID) and email (string) into ctx
// 5. Call next handler

func UserIDFromCtx(ctx context.Context) uuid.UUID
// Exported helper used in handlers to read caller identity
// Panics if called outside an authenticated route (programming error, not user error)

func EmailFromCtx(ctx context.Context) string
// Same pattern for email
```

**Context keys (unexported):**
```go
type contextKey string
const ctxUserID  contextKey = "userID"
const ctxEmail   contextKey = "email"
```

**Depends on:** `golang-jwt/jwt/v5`, `google/uuid`, `model` (not needed — just uuid)

**Do NOT put here:** authorization logic (owner checks belong in handlers), token generation (belongs in auth handler), bcrypt

---

## `backend/middleware/logger.go`

**Purpose:** Structured HTTP request/response logging using `slog`.

**Functions:**
```go
func Logger(next http.Handler) http.Handler
// Wraps ResponseWriter to capture status code
// Logs: method, path, status, latency (time.Since)
// Use slog.Info for 2xx/3xx, slog.Warn for 4xx, slog.Error for 5xx

type statusRecorder struct {
    http.ResponseWriter
    status int
}
func (r *statusRecorder) WriteHeader(code int) { r.status = code; r.ResponseWriter.WriteHeader(code) }
```

**Depends on:** `log/slog` (stdlib), `net/http` (stdlib)

**Do NOT put here:** auth logic, business logging (put that in handlers/store where it occurs)

---

## `backend/handler/helpers.go`

**Purpose:** Shared HTTP utilities used by all handlers. Eliminates repetition.

**Functions:**
```go
func decode(r *http.Request, v any) error
// json.NewDecoder(r.Body).Decode(v)
// Returns error on malformed JSON

func encode(w http.ResponseWriter, status int, v any)
// Sets Content-Type: application/json
// Sets status code
// json.NewEncoder(w).Encode(v)

func writeError(w http.ResponseWriter, status int, message string)
// encode(w, status, map[string]string{"error": message})

func writeValidationError(w http.ResponseWriter, fields map[string]string)
// encode(w, 400, map[string]any{"error": "validation failed", "fields": fields})
```

**Depends on:** `net/http`, `encoding/json` (stdlib only)

**Do NOT put here:** JWT logic, DB queries, request struct definitions (those live in each handler file)

---

## `backend/handler/auth.go`

**Purpose:** Handle user registration and login. Only public endpoints in the system.

**Request/Response types (defined in this file):**
```go
type registerRequest struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type loginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type authResponse struct {
    Token string     `json:"token"`
    User  model.User `json:"user"`
}
```

**Functions:**
```go
func Register(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc
// 1. decode registerRequest
// 2. validate: name, email, password all non-empty
// 3. check email not taken → 409 if duplicate
// 4. bcrypt.GenerateFromPassword(cost=12)
// 5. store.CreateUser(...)
// 6. generate JWT (24h, claims: user_id + email)
// 7. return 201 authResponse

func Login(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc
// 1. decode loginRequest
// 2. validate: email, password non-empty
// 3. store.GetUserByEmail → 401 if not found (don't reveal user exists)
// 4. bcrypt.CompareHashAndPassword → 401 if mismatch
// 5. generate JWT (same as above)
// 6. return 200 authResponse

func generateToken(userID uuid.UUID, email, secret string) (string, error)
// unexported helper
// jwt.NewWithClaims(jwt.SigningMethodHS256, ...)
// Claims: user_id (string), email, exp (now+24h), iat
```

**Depends on:** `store/user.go`, `model`, `middleware` (not needed here), `handler/helpers.go`, `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`

**Do NOT put here:** any authenticated routes, project/task logic, token refresh logic

---

## `backend/handler/projects.go`

**Purpose:** Handle all 5 project endpoints.

**Request types:**
```go
type createProjectRequest struct {
    Name        string  `json:"name"`
    Description *string `json:"description"`
}

type updateProjectRequest struct {
    Name        *string `json:"name"`        // pointer — only update if present
    Description *string `json:"description"`
}
```

**Functions:**
```go
func ListProjects(pool *pgxpool.Pool) http.HandlerFunc
// GET /projects
// caller = middleware.UserIDFromCtx(r.Context())
// store.ListProjects(ctx, pool, callerID)
// return 200 {"projects": [...]}

func CreateProject(pool *pgxpool.Pool) http.HandlerFunc
// POST /projects
// validate name non-empty
// store.CreateProject(ctx, pool, name, desc, callerID)
// return 201 project

func GetProject(pool *pgxpool.Pool) http.HandlerFunc
// GET /projects/:id
// parse chi.URLParam("id") → uuid → 400 if invalid
// store.GetProject → 404 if not found
// return 200 project (with tasks populated)

func UpdateProject(pool *pgxpool.Pool) http.HandlerFunc
// PATCH /projects/:id
// fetch project → 404 if not found
// check project.OwnerID == callerID → 403 if not
// update only non-nil fields
// return 200 updated project

func DeleteProject(pool *pgxpool.Pool) http.HandlerFunc
// DELETE /projects/:id
// fetch project → 404 if not found
// check project.OwnerID == callerID → 403 if not
// store.DeleteProject(ctx, pool, id)
// return 204 (no body)
```

**Depends on:** `store/project.go`, `middleware/auth.go` (for UserIDFromCtx), `handler/helpers.go`, `model`

**Do NOT put here:** task logic, user management, JWT handling

---

## `backend/handler/tasks.go`

**Purpose:** Handle all 4 task endpoints.

**Request types:**
```go
type createTaskRequest struct {
    Title       string     `json:"title"`
    Description *string    `json:"description"`
    Priority    string     `json:"priority"`
    AssigneeID  *uuid.UUID `json:"assignee_id"`
    DueDate     *string    `json:"due_date"`
    // status NOT included — defaults to "todo" on create
}

type updateTaskRequest struct {
    Title       *string    `json:"title"`
    Description *string    `json:"description"`
    Status      *string    `json:"status"`
    Priority    *string    `json:"priority"`
    AssigneeID  *uuid.UUID `json:"assignee_id"`
    DueDate     *string    `json:"due_date"`
}
```

**Functions:**
```go
func ListTasks(pool *pgxpool.Pool) http.HandlerFunc
// GET /projects/:id/tasks
// parse projectID from URL
// parse optional query params: ?status= ?assignee=
// store.ListTasks(ctx, pool, projectID, statusFilter, assigneeFilter)
// return 200 {"tasks": [...]}

func CreateTask(pool *pgxpool.Pool) http.HandlerFunc
// POST /projects/:id/tasks
// validate: title non-empty, priority is valid enum value
// creatorID = callerID from context
// store.CreateTask(ctx, pool, projectID, callerID, input)
// return 201 task

func UpdateTask(pool *pgxpool.Pool) http.HandlerFunc
// PATCH /tasks/:id
// fetch task → 404 if not found
// validate any provided enum fields (status, priority)
// store.UpdateTask(ctx, pool, id, input)
// return 200 task

func DeleteTask(pool *pgxpool.Pool) http.HandlerFunc
// DELETE /tasks/:id
// fetch task → 404 if not found
// fetch project (for OwnerID check)
// check: task.CreatorID == callerID OR project.OwnerID == callerID
// → 403 if neither
// store.DeleteTask(ctx, pool, id)
// return 204

// BONUS (implement if time allows):
func GetProjectStats(pool *pgxpool.Pool) http.HandlerFunc
// GET /projects/:id/stats
// return {"total": n, "by_status": {"todo":n, "in_progress":n, "done":n}}
```

**Depends on:** `store/task.go`, `store/project.go` (for owner check on delete), `middleware/auth.go`, `handler/helpers.go`, `model`

**Do NOT put here:** project CRUD, user logic, any filter logic beyond parsing query params (SQL filtering is in store)

---

## `backend/store/user.go`

**Purpose:** All SQL operations on the `users` table.

**Functions:**
```go
func CreateUser(ctx context.Context, pool *pgxpool.Pool,
    name, email, hashedPassword string) (model.User, error)
// INSERT INTO users ... RETURNING id, name, email, created_at

func GetUserByEmail(ctx context.Context, pool *pgxpool.Pool,
    email string) (model.User, error)
// SELECT id, name, email, password, created_at FROM users WHERE email = $1
// Returns pgx.ErrNoRows if not found (caller maps to 401/404)
```

**Depends on:** `model`, `jackc/pgx/v5`

**Do NOT put here:** bcrypt hashing (belongs in handler), JWT generation, HTTP logic

---

## `backend/store/project.go`

**Purpose:** All SQL operations on the `projects` table.

**Functions:**
```go
func CreateProject(ctx context.Context, pool *pgxpool.Pool,
    name string, description *string, ownerID uuid.UUID) (model.Project, error)
// INSERT INTO projects ... RETURNING *

func ListProjects(ctx context.Context, pool *pgxpool.Pool,
    userID uuid.UUID) ([]model.Project, error)
// SELECT DISTINCT p.* FROM projects p
// LEFT JOIN tasks t ON t.project_id = p.id
// WHERE p.owner_id = $1 OR t.assignee_id = $1
// ORDER BY p.created_at DESC

func GetProject(ctx context.Context, pool *pgxpool.Pool,
    id uuid.UUID) (model.Project, error)
// SELECT * FROM projects WHERE id = $1
// Then: SELECT * FROM tasks WHERE project_id = $1 ORDER BY created_at
// Populates project.Tasks

func UpdateProject(ctx context.Context, pool *pgxpool.Pool,
    id uuid.UUID, name *string, description *string) (model.Project, error)
// Build UPDATE SET only for non-nil fields
// Always returns the updated record

func DeleteProject(ctx context.Context, pool *pgxpool.Pool,
    id uuid.UUID) error
// DELETE FROM projects WHERE id = $1
// CASCADE handles tasks automatically
```

**Depends on:** `model`, `jackc/pgx/v5`, `google/uuid`

**Do NOT put here:** authorization checks (belong in handler), task-specific queries, HTTP encoding

---

## `backend/store/task.go`

**Purpose:** All SQL operations on the `tasks` table.

**Types (defined here):**
```go
type CreateTaskInput struct {
    Title       string
    Description *string
    Priority    string
    AssigneeID  *uuid.UUID
    DueDate     *string
}

type UpdateTaskInput struct {
    Title       *string
    Description *string
    Status      *string
    Priority    *string
    AssigneeID  *uuid.UUID
    DueDate     *string
}
```

**Functions:**
```go
func ListTasks(ctx context.Context, pool *pgxpool.Pool,
    projectID uuid.UUID,
    status *string, assigneeID *uuid.UUID) ([]model.Task, error)
// Build parameterized query dynamically
// Always: WHERE project_id = $1
// Conditionally: AND status = $n, AND assignee_id = $n
// ORDER BY created_at DESC

func CreateTask(ctx context.Context, pool *pgxpool.Pool,
    projectID, creatorID uuid.UUID,
    input CreateTaskInput) (model.Task, error)
// INSERT INTO tasks ... RETURNING *
// status defaults to 'todo' (set in SQL DEFAULT, not in Go)
// updated_at = now() on insert

func GetTask(ctx context.Context, pool *pgxpool.Pool,
    id uuid.UUID) (model.Task, error)
// SELECT * FROM tasks WHERE id = $1

func UpdateTask(ctx context.Context, pool *pgxpool.Pool,
    id uuid.UUID, input UpdateTaskInput) (model.Task, error)
// Build SET clause from non-nil fields only
// Always include: updated_at = now()
// RETURNING *

func DeleteTask(ctx context.Context, pool *pgxpool.Pool,
    id uuid.UUID) error
// DELETE FROM tasks WHERE id = $1

// BONUS:
func GetProjectStats(ctx context.Context, pool *pgxpool.Pool,
    projectID uuid.UUID) (map[string]int, error)
// SELECT status, COUNT(*) FROM tasks WHERE project_id = $1 GROUP BY status
```

**Depends on:** `model`, `jackc/pgx/v5`, `google/uuid`

**Do NOT put here:** project queries (use `store/project.go`), HTTP logic, auth checks

---

## `backend/main.go`

**Purpose:** Entry point. Wires all components together, mounts routes, starts server with graceful shutdown.

**Functions:**
```go
func main()
// 1. config.Load()
// 2. db.NewPool(cfg.DatabaseURL) → log.Fatal on error
// 3. db.RunMigrations(cfg.DatabaseURL)
// 4. r := chi.NewRouter()
// 5. Mount middleware: cors, logger
// 6. r.Post("/auth/register", handler.Register(pool, cfg.JWTSecret))
// 7. r.Post("/auth/login",    handler.Login(pool, cfg.JWTSecret))
// 8. r.Group(func(r chi.Router) {
//        r.Use(middleware.Authenticate(cfg.JWTSecret))
//        // project routes
//        // task routes
//    })
// 9. srv := &http.Server{Addr: ":"+cfg.Port, Handler: r}
// 10. go srv.ListenAndServe()
// 11. wait for SIGINT/SIGTERM
// 12. srv.Shutdown(ctx with 10s timeout)

func setupRouter(pool, cfg) chi.Router  // optional extraction for testability
```

**Depends on:** all other backend packages

**Do NOT put here:** business logic, SQL, JWT parsing — it's a wiring file only

---

## `backend/Dockerfile`

**Purpose:** Multi-stage build producing a minimal production image.

**Stages:**
1. `FROM golang:1.22-alpine AS builder` — `go mod download` then `go build -o /taskflow-api`
2. `FROM alpine:3.19` — copy binary only, add `ca-certificates tzdata`

**Do NOT put here:** `.env` files, source code in runtime stage, `go test` in build (slows CI)

---

# FRONTEND

---

## `frontend/package.json`

**Purpose:** Declare all frontend dependencies.

**Key dependencies:**
```
react, react-dom, react-router-dom
axios
zustand
react-hook-form, @hookform/resolvers, zod
@radix-ui/react-dialog, @radix-ui/react-select (via shadcn)
lucide-react (icons, comes with shadcn)
```

**devDependencies:**
```
typescript, vite, @vitejs/plugin-react
@types/react, @types/react-dom
tailwindcss, postcss, autoprefixer (shadcn requirement)
```

**Do NOT put here:** backend deps, test frameworks unless you're adding bonus tests

---

## `frontend/vite.config.ts`

**Purpose:** Vite build config.

**Key settings:**
```ts
plugins: [react()]
resolve.alias: { "@": path.resolve(__dirname, "./src") }
server.proxy: { "/api": "http://localhost:8080" }  // dev proxy — avoids CORS in dev
```

**Do NOT put here:** env vars (use `.env`), business logic

---

## `frontend/src/types/index.ts`

**Purpose:** All TypeScript interfaces that mirror the backend API shapes. Single source of frontend type truth.

**Types:**
```ts
export interface User {
    id: string;
    name: string;
    email: string;
}

export interface Project {
    id: string;
    name: string;
    description?: string;
    owner_id: string;
    created_at: string;
    tasks?: Task[];
}

export interface Task {
    id: string;
    title: string;
    description?: string;
    status: 'todo' | 'in_progress' | 'done';
    priority: 'low' | 'medium' | 'high';
    project_id: string;
    assignee_id?: string;
    creator_id: string;
    due_date?: string;        // "YYYY-MM-DD"
    created_at: string;
    updated_at: string;
}

export interface AuthResponse {
    token: string;
    user: User;
}

export interface ApiError {
    error: string;
    fields?: Record<string, string>;
}
```

**Depends on:** nothing

**Do NOT put here:** component props types (keep those local to the component), Zustand state types

---

## `frontend/src/store/auth.ts`

**Purpose:** Global auth state. Persisted to localStorage. Single source of truth for "is this user logged in".

**State shape:**
```ts
interface AuthState {
    token: string | null;
    user: User | null;
    setAuth: (token: string, user: User) => void;
    clearAuth: () => void;
    isAuthenticated: () => boolean;  // derived: !!token
}
```

**Implementation notes:**
- Use `zustand` + `zustand/middleware` `persist` with `localStorage`
- Store name: `"auth-storage"` (the localStorage key)
- `isAuthenticated` is a getter function, not a stored value

**Depends on:** `types/index.ts` (User)

**Do NOT put here:** API calls, axios config, routing logic, project/task state (those are local to their pages)

---

## `frontend/src/api/client.ts`

**Purpose:** Axios instance with auth injection and global error handling. All API calls go through this.

**Exports:**
```ts
// Base axios instance (not exported — internal)
const api = axios.create({ baseURL: import.meta.env.VITE_API_URL })

// Request interceptor: reads token from Zustand store, sets Authorization header
// Response interceptor: on 401 → clearAuth() + redirect to /login

// Typed API namespaces:
export const authApi = {
    register(name: string, email: string, password: string): Promise<AuthResponse>
    login(email: string, password: string): Promise<AuthResponse>
}

export const projectsApi = {
    list(): Promise<{ projects: Project[] }>
    create(data: { name: string; description?: string }): Promise<Project>
    get(id: string): Promise<Project>          // includes tasks
    update(id: string, data: Partial<Pick<Project, 'name' | 'description'>>): Promise<Project>
    delete(id: string): Promise<void>
}

export const tasksApi = {
    list(projectId: string, filters?: { status?: string; assignee?: string }): Promise<{ tasks: Task[] }>
    create(projectId: string, data: CreateTaskData): Promise<Task>
    update(id: string, data: UpdateTaskData): Promise<Task>
    delete(id: string): Promise<void>
}
```

**Helper types (defined here):**
```ts
interface CreateTaskData { title: string; description?: string; priority: string; assignee_id?: string; due_date?: string; }
interface UpdateTaskData { title?: string; description?: string; status?: string; priority?: string; assignee_id?: string; due_date?: string; }
```

**Depends on:** `store/auth.ts` (read token), `types/index.ts`

**Do NOT put here:** React state, component logic, URL routing, response caching

---

## `frontend/src/App.tsx`

**Purpose:** Root component. Sets up React Router tree. That's all.

**Structure:**
```tsx
<BrowserRouter>
  <Routes>
    <Route path="/login"    element={<LoginPage />} />
    <Route path="/register" element={<RegisterPage />} />
    <Route element={<ProtectedRoute />}>
      <Route path="/"              element={<ProjectsPage />} />
      <Route path="/projects/:id"  element={<ProjectDetailPage />} />
    </Route>
    <Route path="*" element={<NotFoundPage />} />
  </Routes>
</BrowserRouter>
```

**Depends on:** all pages, `ProtectedRoute`

**Do NOT put here:** global state init, API calls, layout, any business logic — it's purely a routing shell

---

## `frontend/src/components/ProtectedRoute.tsx`

**Purpose:** Guard all authenticated pages. Redirect to login if no token.

**Implementation:**
```tsx
// Read token from Zustand
// if (!token) return <Navigate to="/login" replace />
// return <Outlet />   ← React Router v6 pattern
```

**Depends on:** `store/auth.ts`

**Do NOT put here:** fetching user info, any UI rendering beyond the redirect check

---

## `frontend/src/components/Navbar.tsx`

**Purpose:** Top navigation bar shown on all protected pages.

**Renders:**
- App name / logo (left)
- Logged-in user's name (right)
- Logout button → `clearAuth()` → navigate to `/login`

**Props:** none — reads directly from Zustand

**Depends on:** `store/auth.ts`, `react-router-dom` (useNavigate)

**Do NOT put here:** page-level navigation links (keep it minimal), search, notifications

---

## `frontend/src/pages/LoginPage.tsx`

**Purpose:** Login form with validation and API error handling.

**State:** managed by React Hook Form (no useState for fields)

**Zod schema:**
```ts
z.object({ email: z.string().email(), password: z.string().min(1) })
```

**Behaviour:**
- On valid submit → `authApi.login(...)` → `setAuth(token, user)` → navigate to `/`
- On API error → display message below form (not alert)
- Show loading state on submit button during request
- Link to `/register`

**Depends on:** `api/client.ts` (authApi), `store/auth.ts` (setAuth), `react-hook-form`, `zod`

**Do NOT put here:** project data fetching, auth token storage logic (that's in the store), any routing logic beyond the success redirect

---

## `frontend/src/pages/RegisterPage.tsx`

**Purpose:** Registration form. Mirrors LoginPage pattern.

**Zod schema:**
```ts
z.object({
    name: z.string().min(1),
    email: z.string().email(),
    password: z.string().min(8, "at least 8 characters")
})
```

**Behaviour:**
- On success → `setAuth(token, user)` → navigate to `/`
- Map API `fields` errors to React Hook Form `setError` calls
- Link to `/login`

**Depends on:** same as LoginPage

**Do NOT put here:** anything beyond the registration form flow

---

## `frontend/src/pages/ProjectsPage.tsx`

**Purpose:** Show all accessible projects. Entry point after login.

**Local state:**
```ts
const [projects, setProjects] = useState<Project[]>([])
const [loading, setLoading]   = useState(true)
const [error, setError]       = useState<string | null>(null)
const [showModal, setShowModal] = useState(false)
```

**Behaviour:**
- `useEffect` → `projectsApi.list()` on mount
- Render loading skeleton while fetching
- Render error message if fetch failed
- Render empty state if `projects.length === 0`
- Render `<ProjectCard>` grid
- "New Project" button → `setShowModal(true)`
- On project created → prepend to `projects` state (no refetch)

**Depends on:** `api/client.ts`, `components/ProjectCard.tsx`, `components/CreateProjectModal.tsx`, `types/index.ts`

**Do NOT put here:** task data, project detail logic, filter logic

---

## `frontend/src/components/ProjectCard.tsx`

**Purpose:** Display a single project in the list.

**Props:**
```ts
interface ProjectCardProps {
    project: Project;
    currentUserId: string;       // to show delete button only to owner
    onDelete: (id: string) => void;
}
```

**Renders:**
- Project name (as link to `/projects/:id`)
- Description (or em-dash if empty)
- Created date
- Delete button — only if `project.owner_id === currentUserId`

**Depends on:** `types/index.ts`, `react-router-dom` (Link)

**Do NOT put here:** API calls, modals, task data

---

## `frontend/src/components/CreateProjectModal.tsx`

**Purpose:** Dialog to create a new project.

**Props:**
```ts
interface CreateProjectModalProps {
    open: boolean;
    onClose: () => void;
    onCreated: (project: Project) => void;
}
```

**Behaviour:**
- shadcn `<Dialog>` controlled by `open` prop
- Fields: name (required), description (optional)
- Submit → `projectsApi.create(...)` → `onCreated(project)` → `onClose()`
- Error shown inline
- Loading state on submit button

**Depends on:** `api/client.ts`, `types/index.ts`, shadcn Dialog/Button/Input/Label

**Do NOT put here:** edit logic (separate concern), task creation

---

## `frontend/src/pages/ProjectDetailPage.tsx`

**Purpose:** Show one project's tasks with filters.

**URL params:** `id` from `useParams()`

**Local state:**
```ts
const [project, setProject]       = useState<Project | null>(null)
const [tasks, setTasks]           = useState<Task[]>([])
const [loading, setLoading]       = useState(true)
const [error, setError]           = useState<string | null>(null)
const [statusFilter, setStatus]   = useState<string>('')
const [assigneeFilter, setAssignee] = useState<string>('')
const [taskModal, setTaskModal]   = useState<{ open: boolean; task?: Task }>({ open: false })
```

**Behaviour:**
- On mount: fetch `projectsApi.get(id)` — sets project + tasks
- When filters change: fetch `tasksApi.list(id, { status, assignee })` — updates tasks only
- "New Task" → `setTaskModal({ open: true })` (no task = create mode)
- Click task card → `setTaskModal({ open: true, task })` (task present = edit mode)
- On task created/updated → update tasks array in place or prepend
- On task deleted → filter out from tasks array

**Depends on:** `api/client.ts`, `components/TaskCard.tsx`, `components/TaskModal.tsx`, `components/Navbar.tsx`, `types/index.ts`

**Do NOT put here:** project list logic, bulk operations

---

## `frontend/src/components/TaskCard.tsx`

**Purpose:** Display a single task with inline status change and delete.

**Props:**
```ts
interface TaskCardProps {
    task: Task;
    projectOwnerId: string;
    currentUserId: string;
    onStatusChange: (id: string, status: string) => void;  // triggers optimistic update in parent
    onEdit: (task: Task) => void;
    onDelete: (id: string) => void;
}
```

**Renders:**
- Title, description snippet
- Status badge (colour-coded: todo=gray, in_progress=blue, done=green)
- Priority badge (low=green, medium=yellow, high=red)
- Due date (highlighted red if past)
- Assignee ID (or "Unassigned")
- Status `<Select>` for inline change → calls `onStatusChange`
- Edit button → `onEdit(task)`
- Delete button — only if `task.creator_id === currentUserId || projectOwnerId === currentUserId`

**Depends on:** `types/index.ts`, shadcn Badge/Select/Button

**Do NOT put here:** API calls (all actions delegated to parent via callbacks), modals

---

## `frontend/src/components/TaskModal.tsx`

**Purpose:** Single modal for both create and edit task flows.

**Props:**
```ts
interface TaskModalProps {
    open: boolean;
    onClose: () => void;
    projectId: string;
    task?: Task;                          // undefined = create mode, defined = edit mode
    onSaved: (task: Task) => void;        // called with the created or updated task
}
```

**Behaviour:**
- If `task` present: pre-populate fields, call `tasksApi.update` on submit
- If no `task`: empty form, call `tasksApi.create` on submit
- Fields: title (required), description, status (select, hidden in create mode — defaults to todo), priority (select), assignee_id (text input for now), due_date (date input)
- Zod validation: title required, priority must be valid enum
- Show API field errors inline
- Loading state on submit
- `onClose` on cancel or success

**Depends on:** `api/client.ts`, `types/index.ts`, shadcn Dialog/Button/Input/Label/Select

**Do NOT put here:** project logic, delete action (that's in TaskCard), bulk operations

---

## `frontend/src/pages/NotFoundPage.tsx`

**Purpose:** Catch-all for unknown routes.

**Renders:** "404 — Page not found" with a link back to `/`

**Depends on:** nothing except react-router-dom Link

---

## `frontend/Dockerfile`

**Purpose:** Multi-stage build → Nginx serving the built SPA.

**Stages:**
1. `FROM node:20-alpine AS builder` — `npm ci` then `npm run build` with `VITE_API_URL` as build arg
2. `FROM nginx:alpine` — copy `dist/` → `/usr/share/nginx/html`, copy `nginx.conf`

**Do NOT put here:** dev server, source code in runtime stage

---

## `frontend/nginx.conf`

**Purpose:** Nginx config for the SPA. Critical: must handle client-side routing.

**Key directive:**
```nginx
location / {
    try_files $uri $uri/ /index.html;
}
```

Without this, refreshing `/projects/abc` returns nginx 404 instead of the React app.

**Do NOT put here:** SSL config, proxy rules (not needed — frontend calls API directly at the known URL)

---

# ROOT FILES

---

## `docker-compose.yml`

**Purpose:** Orchestrate all 3 services: postgres, api, frontend.

**Services:**
```yaml
postgres:
  - image: postgres:15-alpine
  - env: POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD
  - volume: pgdata
  - healthcheck: pg_isready

api:
  - build: ./backend
  - env_file: .env
  - ports: 8080:8080
  - depends_on: postgres (condition: service_healthy)

frontend:
  - build: ./frontend (with VITE_API_URL build arg)
  - ports: 3000:80
  - depends_on: api
```

**Do NOT put here:** secrets in plain text, hardcoded passwords

---

## `.env.example`

**Purpose:** Template for all required environment variables. Committed to git. `.env` is never committed.

**All vars:**
```
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=taskflow
POSTGRES_USER=taskflow
POSTGRES_PASSWORD=taskflow_secret
DATABASE_URL=postgres://taskflow:taskflow_secret@postgres:5432/taskflow?sslmode=disable
JWT_SECRET=change_me_in_production_must_be_32_chars_min
API_PORT=8080
VITE_API_URL=http://localhost:8080
```

**Do NOT put here:** real secrets, production values, `.env` itself (gitignore it)

---

## `.gitignore`

**Additions needed:**
```
.env
backend/taskflow-api    # compiled binary
frontend/node_modules/
frontend/dist/
```

---

## `README.md`

**Purpose:** Evaluated as part of the rubric. Must cover all 7 required sections.

**Sections:**
1. Overview — what it is, tech stack table
2. Architecture Decisions — layered approach, tradeoffs, intentional omissions
3. Running Locally — exact commands from `git clone` to browser
4. Running Migrations — "automatic on API startup"
5. Test Credentials — `test@example.com` / `password123`
6. API Reference — table of all 9 endpoints with request/response shapes
7. What You'd Do With More Time — honest list

**Do NOT put here:** vague setup steps, aspirational features as if they exist, apologies

---

# Dependency Graph Summary

```
model ← (no deps)
config ← (no deps)

db ← config, model
store/* ← model, pgx

middleware/auth ← uuid, jwt
middleware/logger ← slog

handler/helpers ← (stdlib only)
handler/auth ← store/user, handler/helpers, jwt, bcrypt
handler/projects ← store/project, middleware, handler/helpers
handler/tasks ← store/task, store/project, middleware, handler/helpers

main ← all of the above

---

types ← (no deps)
store/auth ← types (User)
api/client ← store/auth, types

ProtectedRoute ← store/auth
Navbar ← store/auth
LoginPage / RegisterPage ← api/client, store/auth
ProjectCard ← types
CreateProjectModal ← api/client, types
TaskCard ← types
TaskModal ← api/client, types
ProjectsPage ← api/client, ProjectCard, CreateProjectModal
ProjectDetailPage ← api/client, TaskCard, TaskModal, Navbar

App ← all pages + ProtectedRoute
```
