# TaskFlow — Requirements Traceability Matrix

> Every assignment requirement mapped to its exact implementation location.
> Use this as a pre-submission checklist. Mark each row ✅ when verified.

---

## Part 1 — Authentication

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| `POST /auth/register` endpoint | `handler/auth.go → Register()` | Chi router mounts at `r.Post("/auth/register", ...)` in `main.go` | **Auto-disqualifier**: auth flows break entirely |
| `POST /auth/login` endpoint | `handler/auth.go → Login()` | Chi router mounts at `r.Post("/auth/login", ...)` in `main.go` | **Auto-disqualifier** |
| Passwords hashed with bcrypt | `handler/auth.go → Register()` | `bcrypt.GenerateFromPassword([]byte(password), 12)` — cost hardcoded to 12 in the call site | Cost < 12 = auto-disqualifier |
| bcrypt cost ≥ 12 | `handler/auth.go → Register()` | Second arg to `GenerateFromPassword` is `12`. Add a compile-time constant `const bcryptCost = 12` so it's visible in review | Subtle — reviewer will grep for the cost value |
| JWT expiry: 24 hours | `handler/auth.go → generateToken()` | `exp: time.Now().Add(24 * time.Hour).Unix()` in the JWT claims map | Tokens that never expire or expire too soon |
| JWT claims include `user_id` | `handler/auth.go → generateToken()` | `claims["user_id"] = userID.String()` | Middleware can't read user identity — all protected routes break |
| JWT claims include `email` | `handler/auth.go → generateToken()` | `claims["email"] = email` | Logging won't have email; minor but visible in code review |
| JWT secret from env (not hardcoded) | `config/config.go → Load()` + `handler/auth.go` | `cfg.JWTSecret` passed in at startup; `log.Fatal` if empty | **Auto-disqualifier**: hardcoded secret = immediate rejection |
| All non-auth endpoints require Bearer token | `middleware/auth.go → Authenticate()` + `main.go` | Protected routes mounted inside `r.Group(func(r) { r.Use(Authenticate(secret)) })` | All protected routes become public |
| 401 for missing / invalid token | `middleware/auth.go → Authenticate()` | Returns `writeError(w, 401, "unauthorized")` before calling next | Rubric explicitly checks 401 ≠ 403 |

---

## Part 2 — Projects API

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| `GET /projects` — owned or has tasks in | `store/project.go → ListProjects()` | `SELECT DISTINCT p.* FROM projects p LEFT JOIN tasks t ON t.project_id = p.id WHERE p.owner_id = $1 OR t.assignee_id = $1` | Returns only owned projects — fails the "or has tasks in" requirement |
| `POST /projects` — owner = current user | `handler/projects.go → CreateProject()` | `ownerID = middleware.UserIDFromCtx(r.Context())` passed to store | Owner field wrong — breaks all ownership checks |
| `GET /projects/:id` — detail + tasks | `store/project.go → GetProject()` | Fetch project then `SELECT * FROM tasks WHERE project_id = $1` — populates `project.Tasks` | Tasks not included — frontend ProjectDetailPage breaks |
| `PATCH /projects/:id` — owner only | `handler/projects.go → UpdateProject()` | Fetch project → check `project.OwnerID == callerID` → `403` if not | Non-owners can edit any project |
| `DELETE /projects/:id` — owner only, cascades tasks | `handler/projects.go → DeleteProject()` + `000001_init.up.sql` | Ownership check in handler; `ON DELETE CASCADE` on `tasks.project_id` FK handles cascade in DB | Tasks orphaned without CASCADE; or non-owners can delete |
| 403 for wrong owner (not 401) | `handler/projects.go → UpdateProject()` / `DeleteProject()` | After verifying token is valid (401 gate passed), check ownership → `writeError(w, 403, "forbidden")` | **Rubric explicitly checks this**: 403 ≠ 401 — "do not conflate" is in the spec |
| 404 for missing project | `handler/projects.go` | `store.GetProject` returns `pgx.ErrNoRows` → handler maps to `writeError(w, 404, "not found")` | Generic 500 instead of 404 — looks sloppy |
| Response: `Content-Type: application/json` | `handler/helpers.go → encode()` | `w.Header().Set("Content-Type", "application/json")` inside encode — called by all handlers | Mixed content types; some clients break |

---

## Part 3 — Tasks API

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| `GET /projects/:id/tasks` | `handler/tasks.go → ListTasks()` | Mounted at `r.Get("/projects/{id}/tasks", ...)` under auth middleware | Endpoint missing |
| `?status=` filter | `handler/tasks.go → ListTasks()` + `store/task.go → ListTasks()` | Parse `r.URL.Query().Get("status")` → pass as `*string` to store; SQL appends `AND status = $n` if non-nil | Filtering silently ignored — all tasks returned regardless |
| `?assignee=` filter | Same as above | Parse assignee UUID from query param; validate UUID format → 400 if invalid UUID string | Invalid UUID causes SQL error or silent failure |
| `POST /projects/:id/tasks` | `handler/tasks.go → CreateTask()` | `creatorID = UserIDFromCtx(ctx)` set on insert | Tasks created without creator — delete auth broken |
| `PATCH /tasks/:id` — partial update | `handler/tasks.go → UpdateTask()` + `store/task.go → UpdateTask()` | `UpdateTaskInput` has all pointer fields; SQL SET clause built from non-nil fields only; always sets `updated_at = now()` | Overwriting unset fields with zero values — title clears if not in request body |
| `DELETE /tasks/:id` — creator OR project owner | `handler/tasks.go → DeleteTask()` | Fetch task (get `creator_id`, `project_id`) + fetch project (get `owner_id`) → `403` if neither condition met | Only one condition checked — either creators locked out or anyone can delete |
| `updated_at` changes on PATCH | `store/task.go → UpdateTask()` | `SET ..., updated_at = now()` always included in UPDATE query | Stale timestamps — visible in UI and in rubric review |

---

## Part 4 — General API Requirements

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| All responses `Content-Type: application/json` | `handler/helpers.go → encode()` | Header set before `StatusCode` write in every encode call | Browser/client treats body as text — JSON.parse fails |
| 400 with `{"error":"validation failed","fields":{...}}` | `handler/helpers.go → writeValidationError()` + all handlers | Called whenever required field is blank or enum is invalid | Rubric checks error response shape explicitly |
| 401 for unauthenticated | `middleware/auth.go` | Missing or unparseable token → 401 before handler is reached | |
| 403 for unauthorized action | `handler/projects.go`, `handler/tasks.go` | Ownership checks after authentication passes | Conflating 401/403 — minus points on API design rubric |
| 404 with `{"error":"not found"}` | All handlers | `pgx.ErrNoRows` mapped to `writeError(w, 404, "not found")` | |
| Structured logging | `middleware/logger.go` | `slog.Info` with method, path, status, latency on every request | No logging = looks unprofessional in code review |
| Graceful shutdown on SIGTERM | `main.go` | `signal.NotifyContext` + `srv.Shutdown(ctx)` with 10s timeout | Container restart kills in-flight requests; reviewer checks this |

---

## Part 5 — Data Model & Migrations

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| PostgreSQL | `docker-compose.yml` + `db/db.go` | `postgres:15-alpine` service; pgx driver | |
| Schema via migrations (not auto-migrate) | `db/migrations/000001_init.up.sql` + `db/db.go → RunMigrations()` | golang-migrate with `iofs` source; runs on startup | **Auto-disqualifier**: no migrations = immediate rejection |
| Up **and** down migration for each file | `000001_init.up.sql` + `000001_init.down.sql` | Down drops tables in reverse FK order: tasks → projects → users → extension | Partial down = migration tool errors |
| `User`: id uuid, name, email unique, password (bcrypt), created_at | `000001_init.up.sql` | Defined exactly per spec | Missing `email UNIQUE` causes silent duplicate users |
| `Project`: id, name, description (optional), owner_id → User, created_at | `000001_init.up.sql` | `description TEXT` (no NOT NULL), `owner_id UUID NOT NULL REFERENCES users(id)` | |
| `Task`: id, title, description, status enum, priority enum, project_id, assignee_id (nullable), due_date, created_at, updated_at | `000001_init.up.sql` | CHECK constraints for status + priority; all nullable fields nullable; `creator_id` added beyond spec for delete auth | Missing `creator_id` breaks task delete authorization |
| Indexes where appropriate | `000001_init.up.sql` | 4 indexes: `tasks(project_id)`, `tasks(assignee_id)`, `tasks(project_id, status)`, `projects(owner_id)` | Missing indexes = data modeling rubric deduction |
| Seed: 1 user with known password | `000002_seed.up.sql` | `test@example.com` / `password123`; bcrypt hash pre-generated via `scripts/hashpw/` | Reviewer can't log in without registering — fails "zero manual steps" |
| Seed: 1 project | `000002_seed.up.sql` | Fixed UUID project owned by seed user | |
| Seed: 3 tasks with different statuses | `000002_seed.up.sql` | One each of `todo`, `in_progress`, `done` | |
| Seed is idempotent | `000002_seed.up.sql` | `INSERT ... ON CONFLICT DO NOTHING` on all rows | `docker compose up` twice fails on second run |

---

## Part 6 — Docker & Infrastructure

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| `docker-compose.yml` at repo root | `docker-compose.yml` | Present at monorepo root; spins up postgres + api + frontend | **Auto-disqualifier** if missing or broken |
| Single `docker compose up` — zero manual steps | `docker-compose.yml` + `db/db.go → RunMigrations()` | Migrations run automatically in API entrypoint; seed runs as migration 000002; no manual psql needed | Most common failure point — test this last |
| Postgres healthcheck before API starts | `docker-compose.yml` | `depends_on: postgres: condition: service_healthy` + `healthcheck: pg_isready` | API starts before DB ready → connection error → container exits → compose fails |
| Postgres credentials via `.env` | `.env.example` + `docker-compose.yml` | `POSTGRES_*` vars in `.env`, referenced as `${POSTGRES_*}` in compose | Hardcoded creds in compose = auto-disqualifier |
| `.env.example` with all variables | `.env.example` | All 9 vars present with safe defaults | Reviewer copies `.env.example` → missing var → app crashes |
| API Dockerfile: multi-stage build | `backend/Dockerfile` | Stage 1: `golang:1.22-alpine AS builder`; Stage 2: `alpine:3.19` with binary only | Single-stage = final image is ~1GB golang image; rubric checks this explicitly |
| Migrations run automatically on container start | `main.go` → `db.RunMigrations()` | Called before `srv.ListenAndServe()` — blocking, fatal on error | Reviewer has to run manual psql command |

---

## Part 7 — Frontend Pages & Views

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| Login / Register with client-side validation | `pages/LoginPage.tsx` + `pages/RegisterPage.tsx` | React Hook Form + zod schema; errors shown inline before API call | Raw form without validation = poor UX score |
| Login / Register: error handling | Both auth pages | API `fields` errors mapped to `setError()` calls in RHF; non-field errors shown at form level | Silent failure on wrong password = blank screen |
| Login / Register: JWT storage | `store/auth.ts` | `setAuth(token, user)` → Zustand `persist` → localStorage | Token lost on refresh → user logged out |
| Projects list: show all accessible projects | `pages/ProjectsPage.tsx` | Calls `projectsApi.list()` → renders `<ProjectCard>` for each | |
| Projects list: button to create new project | `pages/ProjectsPage.tsx` + `components/CreateProjectModal.tsx` | "New Project" button → Dialog → POST /projects → prepends to list | |
| Project detail: tasks listed or grouped | `pages/ProjectDetailPage.tsx` | Flat list with `<TaskCard>` components | |
| Project detail: filter by status | `pages/ProjectDetailPage.tsx` | Status `<Select>` → re-calls `tasksApi.list(id, { status })` | Filters that don't work = rubric deduction |
| Project detail: filter by assignee | `pages/ProjectDetailPage.tsx` | Assignee text input → `tasksApi.list(id, { assignee })` | |
| Task create/edit: modal or side panel | `components/TaskModal.tsx` | shadcn `<Dialog>` — same component for both flows via `task?` prop | |
| Task modal: title, status, priority, assignee, due_date | `components/TaskModal.tsx` | All 5 fields present; status hidden on create (defaults to `todo`) | Missing a field = partial data on tasks |
| Navbar: logged-in user's name | `components/Navbar.tsx` | `useAuthStore().user.name` from Zustand | |
| Navbar: logout button | `components/Navbar.tsx` | `clearAuth()` → `navigate('/login')` | |

---

## Part 8 — UX & State

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| React Router for navigation | `App.tsx` | `<BrowserRouter>` + `<Routes>` wrapping all pages | |
| Auth state persists across page refreshes | `store/auth.ts` | Zustand `persist` middleware writes to localStorage key `"auth-storage"` | On every refresh user is logged out — terrible UX |
| Protected routes redirect to `/login` | `components/ProtectedRoute.tsx` | `if (!token) return <Navigate to="/login" replace />` | Unauthenticated users see project pages; API calls get 401 |
| **Loading states must be visible** | All pages | `loading === true` → render skeleton/spinner; `loading === false && data` → render content | Blank white screen during fetch = rubric deduction |
| **Error states must be visible** | All pages | `error !== null` → render error message with retry option; never silent failure | Silent failures look like broken app |
| **Empty states must be sensible** | `ProjectsPage`, `ProjectDetailPage` | `projects.length === 0` → "No projects yet" message; `tasks.length === 0` → "No tasks match this filter" | `undefined` or blank box renders |
| **Optimistic UI for task status changes** | `components/TaskCard.tsx` + `pages/ProjectDetailPage.tsx` | 1. Save `prev = tasks`; 2. `setTasks(...)` with updated status; 3. `await tasksApi.update(...)`; 4. On catch: `setTasks(prev)` | Status updates feel laggy; revert on error doesn't happen |

---

## Part 9 — Design & Polish

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| Component library | shadcn/ui | Installed via `npx shadcn@latest init`; Dialog, Button, Input, Label, Select, Badge used throughout | Not using a library = more time on raw CSS |
| **Responsive at 375px (mobile)** | All pages + CSS | shadcn uses Tailwind responsive utilities; check each page at 375px explicitly: no horizontal overflow, no overlapping elements | Broken mobile layout = rubric deduction |
| **Responsive at 1280px (desktop)** | All pages | Full-width layouts use max-width containers; grid for project cards | |
| No broken layouts | Final validation pass | Run `npm run build` → check dist; open at both widths in browser | Console errors in prod build = rubric deduction |
| No console errors in production build | Final validation pass | `npm run build` + serve `dist/` via nginx; check DevTools console | |
| Sensible empty states — no `undefined` | All pages | Use optional chaining `?.` + fallback values `?? ''`; never render raw `{task.assignee_id}` without null check | `undefined` renders as literal string on page |

---

## Part 10 — README (Rubric Section)

| Requirement | File / Module | How it is satisfied | Risk of missing it |
|---|---|---|---|
| Section 1: Overview | `README.md` | What it is, what it does, tech stack table | **Auto-disqualifier** if README missing entirely |
| Section 2: Architecture Decisions | `README.md` | Why layered arch; why no ORM; why no React Query; PATCH semantics; what was skipped + why | Rubric explicitly scores this. Generic copy-paste fails |
| Section 3: Running Locally (exact commands) | `README.md` | `git clone` → `cp .env.example .env` → `docker compose up` — every command exact | Reviewer can't start the app |
| Section 4: Running Migrations | `README.md` | "Migrations run automatically on API startup via golang-migrate." | Reviewer manually runs SQL dumps — bad experience |
| Section 5: Test Credentials | `README.md` | `test@example.com` / `password123` — matches seed exactly | Reviewer registers manually; can't verify seed data |
| Section 6: API Reference | `README.md` | Table of all 9 endpoints with method, path, request body, response shape | Reviewer can't test API in isolation |
| Section 7: What You'd Do With More Time | `README.md` | Honest list: rate limiting, refresh tokens, React Query, pagination, drag-and-drop, real-time updates, proper test suite | This section is explicitly evaluated. Shallow answers lose points |

---

## Part 11 — Automatic Disqualifiers Checklist

| Auto-disqualifier | Where prevented | How to verify |
|---|---|---|
| App does not run with `docker compose up` | `docker-compose.yml` + healthcheck + auto-migrate | Run `docker compose down -v && docker compose up --build` from scratch |
| No database migrations | `db/migrations/` + `db/db.go → RunMigrations()` | Check `schema_migrations` table exists after startup |
| Passwords stored in plaintext | `handler/auth.go` → `bcrypt.GenerateFromPassword` | `SELECT password FROM users LIMIT 1` → must start with `$2a$12$` |
| JWT secret hardcoded | `config/config.go` → reads `JWT_SECRET` env; `log.Fatal` if empty | Grep for any string literal containing "secret" near JWT signing |
| No README | `README.md` | File exists with all 7 sections |
| Submission after deadline | N/A | Submit before 72h window closes |

---

## Part 12 — Bonus Requirements

| Bonus Requirement | File / Module | How it is satisfied | Time cost |
|---|---|---|---|
| Pagination on list endpoints | `store/project.go`, `store/task.go`, handlers | `?page=&limit=` → `LIMIT $n OFFSET $m` in SQL; response includes `total` count | ~30 min — do only if ahead of schedule |
| `GET /projects/:id/stats` | `handler/tasks.go → GetProjectStats()` + `store/task.go → GetProjectStats()` | `SELECT status, COUNT(*) GROUP BY status`; return `{total, by_status}` | ~20 min |
| ≥ 3 integration tests | `backend/*_test.go` | Spin up real postgres in test; hit HTTP endpoints; assert status codes | ~45 min |
| Drag-and-drop task status | `components/TaskCard.tsx` or Kanban board | `@dnd-kit/core` or `react-beautiful-dnd` | ~60 min — skip unless everything else is done |
| Dark mode toggle | `App.tsx` + CSS variables + localStorage | shadcn supports dark mode via class; Zustand or localStorage for persistence | ~20 min — easiest bonus |
| Real-time updates (WebSocket/SSE) | Requires backend SSE endpoint + frontend EventSource | `GET /projects/:id/events` streams task changes | ~90 min — skip for this exercise |

---

## High-Risk Items Summary

These are the items most likely to be missed under time pressure:

| Priority | Item | Why it's high risk |
|---|---|---|
| 🔴 CRITICAL | `docker compose up` works cold | Most submissions fail here. Test last, on a clean machine simulation (`docker compose down -v`) |
| 🔴 CRITICAL | bcrypt cost = 12 | Easy to write `bcrypt.DefaultCost` (10) — must be explicit `12` |
| 🔴 CRITICAL | JWT secret from env | Instinct is to put a default string — even as fallback, it's disqualifying |
| 🔴 CRITICAL | Seed data has real bcrypt hash | Using placeholder text in `seed.sql` means login fails |
| 🟠 HIGH | 401 vs 403 separation | Conflating them is a rubric deduction under "API design" |
| 🟠 HIGH | PATCH doesn't overwrite with zero values | Pointer fields everywhere in `updateTaskRequest` |
| 🟠 HIGH | Loading/error/empty states on every page | Easy to ship happy-path only, especially under time pressure |
| 🟠 HIGH | Optimistic update reverts on error | Second half (the catch block revert) is often forgotten |
| 🟠 HIGH | Nginx `try_files` for React Router | Browser refresh on any nested route returns nginx 404 |
| 🟡 MEDIUM | Responsive at 375px | Test explicitly — don't assume Tailwind handles it |
| 🟡 MEDIUM | README Section 7 (What I'd do with more time) | Generic answers score poorly; must reflect your actual tradeoffs |
| 🟡 MEDIUM | `task.creator_id` populated on create | If forgotten, delete auth silently locks everyone out |
