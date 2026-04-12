# TaskFlow

Initial monorepo scaffold for the take-home assignment.

This commit only sets up the planned repo structure, placeholder environment and compose files, and minimal backend/frontend bootstrapping. No business logic has been implemented yet.

Planned stack, per repo decisions:
- Go + chi backend
- React + TypeScript + Vite frontend
- PostgreSQL

Implementation details, migrations, API endpoints, and full local run instructions will be added in later phases.





Implement Phase 2 only: backend auth foundation.

Scope:

model structs needed for auth and shared domain types
user store functions needed for register/login
password hashing utilities for register
JWT generation and validation utilities
auth handlers for:
POST /auth/register
POST /auth/login
auth middleware for protected routes
structured JSON error responses for validation and unauthorized cases

Requirements:

bcrypt cost >= 12
JWT expiry 24 hours
JWT claims must include user_id and email
JSON responses only
400 validation failed with fields object
401 for unauthenticated requests
no project/task handlers yet
no extra architecture or service layer
keep code simple and aligned with the plan

Important:

do not require unrelated config at startup before it is actually used
keep startup wiring minimal and phase-appropriate
summarize created files, request/response shapes, and manual tests to run next


At the end, summarize:
- routes added
- request/response shapes
- files changed
- what I should test manually now




Phase 4:
Implement only the Projects API.

Required endpoints:
- GET /projects
- POST /projects
- GET /projects/:id
- PATCH /projects/:id
- DELETE /projects/:id

Rules:
- owner = current authenticated user on create
- update/delete owner only
- GET /projects should list projects the user owns or has tasks in
- GET /projects/:id should include project details and its tasks

Requirements:

All routes protected by auth middleware
Project owner = authenticated user on create
Only owner can update/delete project
GET /projects returns projects owned by user
GET /projects/ returns project details
include created_at and updated_at
validation errors return 400 with fields object
unauthorized access returns 403
not found returns 404

Keep error handling aligned with the assignment:
- 401 unauthenticated
- 403 unauthorized
- 404 not found
- 400 validation failed with fields object

Do NOT implement tasks yet.
Do NOT add service layer.
Keep handler → store structure.
Return consistent JSON responses.

Output:

created files
request/response shapes
manual tests to run

Do not over-engineer. This is a take-home assignment optimized for clarity, correctness, reviewability, and completion within 4–5 hours.


phase 5:
Implement only the Tasks API.

Required endpoints:
- GET /projects/:id/tasks
- POST /projects/:id/tasks
- PATCH /tasks/:id
- DELETE /tasks/:id

Requirements:
- support filters ?status= and ?assignee=
- PATCH should allow updating title, description, status, priority, assignee, due_date
- DELETE allowed only for project owner or task creator
- keep structured JSON errors
- keep handlers/services/db code simple and readable

Do not change architecture without explicit justification.
At the end, list the manual API tests I should run.


phase 6:
Read the planning files and implement only frontend auth and routing.

Scope:
- React + TypeScript + Vite
- React Router
- login page
- register page
- auth store
- JWT persistence across refresh
- protected routes redirecting to /login
- navbar with logged-in user and logout
- visible loading and error states

Do not implement project/task UI yet.
Do not introduce extra state libraries beyond the planned choice.
Keep components small and readable.

phase 7:
Implement only the remaining core frontend views.

Required:
- Projects list page
- create project action
- Project detail page
- tasks displayed in a sensible grouped or listed way
- filters for status and assignee
- task create/edit modal or side panel
- loading, error, and empty states
- responsive behavior for mobile and desktop
- no silent failures

Also implement optimistic UI for task status changes:
- update immediately in UI
- revert on API error
- show visible feedback on failure

Do not add drag-and-drop unless everything else is already done.

phase 8:
Implement only infrastructure and documentation polish.

Scope:
- backend multi-stage Dockerfile
- frontend Dockerfile
- docker-compose.yml that starts db, backend, frontend
- .env.example with all required variables
- migration execution strategy
- seed setup integration
- README with these sections:
  1. Overview
  2. Architecture Decisions
  3. Running Locally
  4. Running Migrations
  5. Test Credentials
  6. API Reference
  7. What I'd Do With More Time

The result should be reviewer-friendly and aligned with the assignment.
Do not invent unsupported commands.