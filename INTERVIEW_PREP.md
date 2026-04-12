# Code Review Interview — Talking Points

> These are concise, natural answers for the 30-minute code review call.
> Read them once. Don't memorize them — understand them.

---

## Why a monorepo?

*"For a project this size, a monorepo is just simpler. Backend and frontend are deployed together via one `docker compose up`, they share the same `.env`, and there's no overhead of managing cross-repo PRs or versioning a shared API contract. If this grew into a multi-team product with independent deployment cycles, I'd split them. But for a single-team full-stack project, co-location is the right call."*

---

## Why Go + chi?

*"Go was specified as preferred, and it's genuinely a good fit here — fast startup, low memory footprint, and the HTTP stdlib is solid enough that you don't need much on top of it. I picked chi over something like gin because it's just a router — it wraps `net/http` natively, uses standard `http.Handler` interfaces, and doesn't invent its own context or middleware signature. That means every middleware I wrote is drop-in compatible with any other Go HTTP code, and it keeps the handler layer very readable."*

---

## Why migrations instead of auto-migrate?

*"Auto-migrate — whether that's GORM's `AutoMigrate` or similar — is fine for prototypes, but it's a footgun in anything real. It'll add columns or tables, but it won't drop columns, rename them, or make destructive changes safely. You lose visibility into what the schema actually looks like at any point in time, and you can't roll back cleanly. With golang-migrate and explicit `.up.sql` / `.down.sql` files, the schema history is in version control, every change is reviewable, and the down migration is a first-class thing I have to think about. That discipline matters — especially when the first thing a reviewer checks is whether your migrations are real."*

---

## Why was `task.creator_id` added?

*"The assignment says 'project owner or task creator can delete a task.' That's two different identity checks. Without `creator_id` on the task, you can only check project ownership — you'd have to deny deletion to anyone who isn't the project owner, which contradicts the spec. So I added `creator_id` at the schema level from day one, populated it from the JWT on task creation, and check both conditions in the delete handler. It's a minor schema addition but it's what makes the authorization rule actually work."*

---

## How are authentication and authorization separated?

*"Authentication is the middleware layer — it just answers 'who are you?' It reads the `Authorization` header, validates the JWT, and injects `userID` and `email` into the request context. It returns `401` if the token is missing or invalid. It knows nothing about resources or ownership.*

*Authorization is in the handlers — 'are you allowed to do this?' After the middleware confirms identity, the handler fetches the resource, checks if the caller is the owner, and returns `403` if not. Those are deliberately different HTTP status codes for a reason: `401` means re-authenticate, `403` means you're authenticated but not permitted. The spec explicitly says not to conflate them, and most implementations I've seen do conflate them."*

---

## Why Zustand for state management?

*"The two obvious choices were Redux Toolkit and Zustand. Redux would be overkill for the auth slice this app needs — I'd be writing reducers and actions for storing a JWT. Zustand lets me define the entire auth store in about 15 lines, and the `persist` middleware serializes it to localStorage automatically, so auth survives page refreshes without any extra wiring. I deliberately kept server data — projects, tasks — in local component state with `useState`, because there's no shared cross-component state to manage there. Adding Zustand for that would be unnecessary abstraction."*

---

## Why shadcn/ui?

*"shadcn is different from a traditional component library — it doesn't ship a package you install and import from. It generates component source directly into your project, so you own the code and can modify it freely. That's good for a take-home where you want to move fast but also want the UI to look polished. The components are built on Radix UI primitives for accessibility, and they use Tailwind for styling, so they compose well without fighting the library. The practical reason: I can get a production-looking Dialog, Select, and Badge working in minutes rather than building them from scratch."*

---

## What did you intentionally not build?

*"A few things, all deliberate:*

*Refresh tokens — the spec asks for 24-hour JWT expiry, which is what the assignment needs. Adding refresh tokens would mean either a token store in the DB (now it's stateful) or a separate short-lived/long-lived token pair. That's real complexity for zero benefit in this scope.*

*Rate limiting — it's not in the requirements. I'd mention it in 'what I'd add next.'*

*A service layer — there's no business logic that justifies an extra abstraction between the handler and the store. Adding a 'service' that just calls the store function is ceremony, not value. If this app grew, the first thing I'd do is pull complex business rules into that layer.*

*Repository interfaces — I'm not writing two implementations of the store, so an interface would be untestable theatre. If I were writing unit tests that mocked the DB, I'd add the interface then.*

*React Query — `useState` + `useEffect` is simpler to read and explain in a 30-minute review. React Query is more production-correct for caching and background refetch, but it's an extra API surface I'd have to justify during review."*

---

## What would you improve with more time?

*"Honestly, a few things I cut for time:*

*Test coverage — I'd add integration tests that spin up a real Postgres in CI and test the full request/response cycle. Unit tests on pure CRUD handlers are low signal; integration tests on the auth flow and task deletion logic are where the interesting edge cases live.*

*Proper assignee UX — right now you enter a UUID in the assignee field. In a real product that would be a user search or dropdown. I'd add a `GET /projects/:id/members` endpoint that returns users with access, and the frontend would use that to populate the picker.*

*Pagination — the list endpoints return everything. For small datasets that's fine; for any real deployment you'd want `?page=&limit=` on projects and tasks.*

*Optimistic updates more broadly — I implemented it for status changes on tasks, but edits via the modal do a full refetch. I'd extend the optimistic pattern to all mutations.*

*Error boundaries in React — right now an uncaught render error would crash the whole page. I'd add an ErrorBoundary at the route level to degrade gracefully."*

---

## Bonus: If they ask "what was the hardest part?"

*"The trickiest part was the task delete authorization. You need to check two things — is the caller the task creator, or are they the project owner? That means one DB call to get the task, a second to get the project's owner_id, and then a boolean check across both. It's not hard, but it's the kind of thing that's easy to implement only half of — checking just one condition and calling it done. I made sure both branches are covered and added `creator_id` to the schema to make it possible in the first place."*

---

## Bonus: If they ask "why not use an ORM?"

*"The schema here is simple enough that raw SQL is more readable than whatever the ORM generates. With pgx I know exactly what query hits the database — there's no magic. For the dynamic filter on `GET /projects/:id/tasks`, building a parameterized query with conditional WHERE clauses in SQL is three lines. With an ORM you're chaining `.Where()` calls that produce unpredictable queries under the hood. And for this assignment specifically, the reviewer can see exactly what SQL runs — that's a better signal of DB competence than 'I called `.Find()`'."*
