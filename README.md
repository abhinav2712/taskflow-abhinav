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


Do not over-engineer. This is a take-home assignment optimized for clarity, correctness, reviewability, and completion within 4–5 hours.





-----

Implemented Phase 2 only: auth models, user store, bcrypt/JWT auth flow, JSON helpers, auth middleware, and minimal router wiring for the two public auth routes.

Routes Added

POST /auth/register
POST /auth/login
The auth middleware is implemented in backend/middleware/auth.go, but I did not mount any protected business routes yet since that belongs to later phases.

Request / Response Shapes

POST /auth/register
Request:
{ "name": "Test User", "email": "test@example.com", "password": "password123" }


sucess 201:
{
  "token": "<jwt>",
  "user": {
    "id": "<uuid>",
    "name": "Test User",
    "email": "test@example.com",
    "created_at": "<timestamp>"
  }
}

POST /auth/login


Request:
{ "email": "test@example.com", "password": "password123" }
Success 200: same shape as register

Validation error 400
{ "error": "validation failed", "fields": { "email": "is required" } }


Bad JSON body 400:
{ "error": "invalid request body" }


Unauthenticated 401:

{ "error": "unauthorized" }

duplicate email 409:

{ "error": "email already in use" }

Files Changed

Added backend/model/models.go
Added backend/store/user.go
Added backend/handler/helpers.go
Added backend/handler/auth.go
Added backend/middleware/auth.go
Added backend/middleware/logger.go
Updated backend/cmd/server/main.go
Updated backend/go.mod

What I Actually Added, In Plain English

    models.go: plain Go structs for User, Project, and Task. Think of these as the shared data shapes our backend passes around. Password has json:"-" so it never leaks into API responses.

    store/user.go: the DB-only layer for auth. It knows how to create a user and fetch a user by email, but it knows nothing about HTTP or JWT.

    handler/helpers.go: small shared HTTP helpers so every endpoint returns consistent JSON.

    handler/auth.go: register/login endpoints. This is where request validation, bcrypt hashing, password checking, and JWT creation happen.

    middleware/auth.go: reads Authorization: Bearer ..., validates the token, and stores user_id and email in the request context for later protected routes.

    main.go: now starts an actual HTTP server, mounts the two auth routes, and keeps startup minimal.

Important Auth Decisions Kept Aligned To The Plan

    bcrypt cost is explicitly 12 in backend/handler/auth.go

    JWT expiry is 24 * time.Hour

    JWT claims include user_id and email

    login returns 401 for wrong email or wrong password

    validation failures return 400 with a fields object

    no service layer added

    no project/task handlers added

What You Should Test Manually Now
1. Register a brand-new user.
Bash

curl -i -X POST http://localhost:8080/auth/register \
 -H 'Content-Type: application/json' \
 -d '{"name":"Alice","email":"alice@example.com","password":"password123"}'

2. Log in with that user.
Bash

curl -i -X POST http://localhost:8080/auth/login \
 -H 'Content-Type: application/json' \
 -d '{"email":"alice@example.com","password":"password123"}'

3. Log in with the seeded Phase 1 user.
Bash

curl -i -X POST http://localhost:8080/auth/login \
 -H 'Content-Type: application/json' \
 -d '{"email":"test@example.com","password":"password123"}'

4. Try missing fields and confirm you get 400 plus fields.
Bash

curl -i -X POST http://localhost:8080/auth/login \
 -H 'Content-Type: application/json' \
 -d '{"email":""}'

5. Try the wrong password and confirm you get 401.
Bash

curl -i -X POST http://localhost:8080/auth/login \
 -H 'Content-Type: application/json' \
 -d '{"email":"test@example.com","password":"wrong"}'

6. Try duplicate registration and confirm you get 409.
7. Try malformed JSON and confirm you get 400 {"error":"invalid request body"}.

    One note: I couldn't run the Go server in this environment because go isn't installed here, so this was implemented and checked by code inspection rather than a live boot.



