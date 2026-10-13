# Architecture

## Stack

| Concern | Choice | Notes |
|---|---|---|
| Frontend | React + TypeScript + Vite | |
| Routing | React Router v7 | Client-side, URL-driven navigation |
| UI components | Mantine | Attractive defaults, full-featured, no custom theming required |
| Data fetching | TanStack Query | Server state, caching, loading/error handling |
| Backend | Go | Modular monolith; standard library HTTP server |
| Database | PostgreSQL 18 | Explicit SQL, no heavy ORM |
| File storage | S3-compatible | Local development store to be chosen in Milestone 3; any S3-compatible service in production |
| Migrations | golang-migrate | SQL files only, no down migrations |
| Auth | Session-based | Multi-user with shared access; users created via CLI; no user-management UI for MVP |
| Dev environment | Docker Compose | |
| Testing | Go integration tests (backend); lint and build (frontend) | Backend tests run against a real PostgreSQL database. Playwright is deferred until the UI stabilises |

Go and Node versions are pinned in `.tool-versions`. `backend/go.mod` carries the same Go version, and CI reads both files.

## Repo Structure

Entries marked *(planned)* are part of the agreed design but do not exist yet.

```
doc-tracker/
├── CLAUDE.md                    # instructions for AI-assisted development
├── README.md
├── LICENSE                      # MPL 2.0
├── docs/                        # architecture, domain model, api conventions, milestones
├── frontend/
│   ├── src/
│   │   ├── features/
│   │   │   ├── auth/
│   │   │   ├── cases/
│   │   │   ├── people/
│   │   │   ├── claim-lines/
│   │   │   ├── applications/    # (planned)
│   │   │   ├── life-events/
│   │   │   ├── documents/
│   │   │   ├── trash/
│   │   │   └── exports/         # (planned)
│   │   ├── components/          # (planned) shared UI only; first occupant is the date input
│   │   ├── api/                 # typed API client
│   │   └── lib/                 # (planned) pure utilities, no business logic
│   ├── index.html
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── backend/
│   ├── cmd/
│   │   ├── server/
│   │   │   └── main.go          # wires everything
│   │   ├── create-user/         # CLI: create a user
│   │   └── seed/                # CLI: load sample cases
│   ├── internal/
│   │   ├── auth/
│   │   ├── users/
│   │   ├── activitylog/
│   │   ├── cases/
│   │   ├── people/
│   │   ├── claimlines/
│   │   ├── applications/        # (planned)
│   │   ├── lifeevents/
│   │   ├── documents/
│   │   ├── trash/
│   │   ├── storage/             # (planned)
│   │   ├── exports/             # (planned)
│   │   └── testhelpers/         # test database setup shared by integration tests
│   ├── db/
│   │   ├── migrations/          # numbered SQL files, e.g. 001_initial_schema.up.sql
│   │   └── schema.sql           # generated snapshot of the current schema
│   └── platform/
│       ├── server.go            # HTTP server setup, middleware chain
│       ├── config.go            # env-based config structs
│       ├── database.go          # connection pool setup
│       ├── migrate.go           # runs migrations on startup
│       └── ...                  # request parsing and response helpers
│
├── docker-compose.yml
├── docker-compose.dev.yml       # dev overrides
├── Makefile
├── .tool-versions
└── .env.example
```

## Backend Package Structure

Each module under `internal/` follows the same layout:

```
internal/<module>/
├── handler.go      # HTTP handlers, request parsing, response writing
├── service.go      # units of work: transaction, change, activity log entry
├── store.go        # DB interface definition + SQL implementation
└── model.go        # domain types for this module
```

- Handlers read the request and write the response. They know nothing about SQL or transactions.
- Services decide what happens as one unit of work. They start the transaction, make the change through the store, write the activity log entry, and commit. They know nothing about HTTP.
- Stores run SQL. They know nothing about business rules or transaction boundaries.

The structure is consistent regardless of module size. This makes AI-assisted development predictable — adding a field or endpoint always touches the same set of files in the same locations.

## Transactions

- Every write runs inside a database transaction started by the service.
- The activity log entry for a write is inserted in the same transaction. If either the change or the log entry fails, neither is saved.
- A user-initiated save is one unit of work. Changing a document's fields, parent, and phase statuses in one save commits or rolls back together.
- Stores do not begin or commit transactions. They run their SQL on whatever transaction the service has opened.
- Calls to object storage are made outside database transactions. A transaction is never held open while a file is uploaded, downloaded, or deleted.

## Module Responsibilities

| Module | Owns | Notes |
|---|---|---|
| `auth` | Session management, auth middleware | Imports `users` |
| `users` | User records, password hashing | Imports nothing internal |
| `activitylog` | ActivityLog entries | Imports nothing internal; imported by every module that writes |
| `cases` | Case lifecycle and status; whether a Case is active | |
| `people` | Person records, PersonRelationship; whether a Person is active | |
| `claimlines` | ClaimLine records and status | |
| `applications` *(planned)* | Application records and their applicants; deriving an Application's Documents from each applicant's path to the root ancestor, its status, and copies required | Uses `people`, `cases`, and `documents` through interfaces |
| `lifeevents` | LifeEvent records, including shared marriages; whether a LifeEvent is active | |
| `documents` | Document records, DocumentNamedPerson, FileAttachment, readiness; also hosts `GET /api/v1/document-statuses` until custom status management warrants its own package | Uses `storage` through an interface *(planned)* |
| `trash` | Trash listing, the 30-day purge, CaseDeletionRecord; coordinates restore and permanent delete | Owns no entity tables. Asks each owning module to restore or delete its own records |
| `storage` *(planned)* | File upload/download, object storage abstraction | Imports nothing internal |
| `exports` *(planned)* | ZIP assembly, generated file names, the manifest, export downloads | Uses `applications`, `documents`, and `storage` through interfaces |

## Module Boundaries

Three rules govern how modules relate to each other's data.

1. **A module is the only writer of its own tables.** No other module inserts, updates, or deletes its rows. When a unit of work spans modules, such as saving a Person together with their birth event, or promoting a spouse to a Person, the service running it asks the owning module to make its part of the change. All parts run in the same transaction.
2. **A module owns the rules about its own entities.** Whether a Person, LifeEvent, or Case is active, meaning it exists, belongs to the Case, is not trashed, and is not frozen under a trashed parent, is answered by the owning module and nowhere else. Other modules ask; they do not re-derive the answer in their own SQL.
3. **Reading across tables in SQL is allowed.** A store may join to another module's tables to assemble what it returns, such as a document list with its status labels. Reads carry no rules of their own, so nothing is duplicated.

### How modules call each other

Feature modules do not import each other. A module that needs something from another declares a small interface describing what it needs, and `cmd/server/main.go` supplies the owning module's implementation at startup.

```go
// in package documents
type PersonChecker interface {
    RequireActive(ctx context.Context, caseID, personID string) error
}
```

This keeps each rule with a single owner without creating import cycles, which Go does not permit. Cycles would otherwise be unavoidable: `people` asks `lifeevents` to write a birth event, and `lifeevents` asks `people` whether a Person is active.

`trash` follows the same rules. It lists what is trashed and decides when to purge, but it restores and permanently deletes by asking each owning module, because deletion carries entity-specific rules such as the handling of shared marriages.

## Dependency Direction

Package imports:

```
platform     ← imported by every module; imports nothing internal
users        ← imports nothing internal
auth         ← imports users
activitylog  ← imports nothing internal
cases, people, claimlines, lifeevents, documents, trash, applications (planned)
             ← import auth (current user) and activitylog
storage      ← imports nothing internal                      (planned)
documents    ← additionally imports storage's interface      (planned)
exports      ← imports storage's interface                   (planned)
```

No cycles. Everything else a module needs from another arrives through an interface wired in `cmd/server/main.go`, which is the only package that imports every module.

## Key Constraints

- `storage` never imports `documents`. Storage knows about bytes and object keys only.
- `documents` uses `storage` via an interface, keeping storage testable and swappable.
- `platform/` is pure infrastructure — server setup, config, DB pool, transaction helper. No business logic.
- Frontend feature structure mirrors backend module structure. A feature change touches predictable files in both.

## Division of Responsibility

The backend is the single authority for domain rules and derived facts. A rule lives in one place. The backend and frontend share no code, so a rule held on both sides would exist twice and drift apart.

The frontend owns presentation and interaction state, including the sorting, grouping, and formatting that presentation needs.

| Backend | Frontend |
|---|---|
| Domain rules and the facts derived from them: document readiness and its reasons, the stage a document is waiting on, case progress counts, the birth window implied by a recorded age, the comparison of recorded details for a person | How those facts are shown: grouping, sorting, filtering, and formatting values such as "about 1890" |
| Validation, as the final authority | Interaction state: what is typed in a form, which view is selected |

Two exceptions:

- **Input format checks may be mirrored in the browser** so the user gets immediate feedback. The backend still validates and has the final say.
- **A derived value needed live, on input that has not been saved, is decided case by case.** The options are to show it only after saving, to add a preview request, or to duplicate that one calculation knowingly.

## Frontend

- `api/` holds the typed API client. Lists inside a case arrive complete in one response, so screens always work with a case's full data.
- Components shared by more than one feature live in `components/`. Anything used by a single feature stays in that feature's folder.
- Within a feature folder, each form, modal, and section is its own component file. The page file arranges them and holds no form or modal markup of its own. Existing pages that predate this rule are split when they are next changed.
- Child entities are managed inside their parent's modal, not on separate pages.

## Deployment Portability

The app is intended to be self-hostable on any stack. To preserve this:

- All infrastructure configuration (database, storage, ports) is read from environment variables. No provider-specific values are hardcoded.
- The `storage` module speaks the S3 API only. It must remain behind an interface so alternative backends can be substituted without touching application code.
- Deployment reference files (e.g. `Caddyfile`, `docker-compose.prod.yml`) live in the repo as examples, not requirements.
- No Hetzner-specific, Cloudflare-specific, or any other provider-specific managed services are used inside the application layer.

The reference production stack (Hetzner VPS + Caddy + Cloudflare R2) is one valid deployment target, not the only one.

## Production Shape

- The backend serves the built frontend, so the application deploys as a single image. HTTPS is terminated in front of it by a reverse proxy.
- Sessions are stored in the database, so they survive restarts and deploys.
- The create-user and password-reset commands ship inside the image.
- Object storage has encryption at rest enabled. The reference deployment shows how.

## Migration Convention

- Tool: `golang-migrate`
- Files: numbered SQL with `.up.sql` suffix, e.g. `001_initial_schema.up.sql`, `002_users.up.sql`
- No down migrations (`.down.sql` files are not used)
- A migration must never discard data. A change that would lose data is written as a migration that preserves or converts it.
- Take a database backup before every upgrade. Migrations only run forwards, so a backup is the only way back.
- `db/schema.sql` is a generated snapshot of the full schema, produced by `make schema` from a freshly migrated database. It is never edited by hand, and CI fails if it does not match the migrations.
