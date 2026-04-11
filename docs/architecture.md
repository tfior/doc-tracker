# Architecture

## Stack

| Concern | Choice | Notes |
|---|---|---|
| Frontend | React + TypeScript + Vite | |
| Backend | Go | Modular monolith |
| Database | PostgreSQL | Explicit SQL, no heavy ORM |
| File storage | S3-compatible | MinIO locally, S3 in production |
| Migrations | golang-migrate | SQL files only, no down migrations |
| Auth | Session-based | Single-owner MVP, no user management |
| Dev environment | Docker Compose | |
| Testing | Playwright (frontend), Go tests (backend) | |

## Repo Structure

```
doc-tracker/
├── docs/                        # architecture, domain model, api conventions, CLAUDE.md
├── frontend/
│   ├── src/
│   │   ├── features/
│   │   │   ├── cases/
│   │   │   ├── people/
│   │   │   ├── claim-lines/
│   │   │   ├── life-events/
│   │   │   ├── documents/
│   │   │   └── exports/
│   │   ├── components/          # shared UI only (buttons, modals, layout)
│   │   ├── api/                 # typed API client
│   │   └── lib/                 # pure utilities, no business logic
│   ├── index.html
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go          # wires everything, ~30 lines
│   ├── internal/
│   │   ├── auth/
│   │   ├── cases/
│   │   ├── people/
│   │   ├── claimlines/
│   │   ├── lifeevents/
│   │   ├── documents/
│   │   ├── storage/
│   │   └── exports/
│   ├── db/
│   │   ├── migrations/          # numbered SQL files, e.g. 001_initial_schema.sql
│   │   └── schema.sql           # current schema snapshot
│   └── platform/
│       ├── server.go            # HTTP server setup, middleware chain
│       ├── config.go            # env-based config structs
│       └── database.go          # connection pool setup
│
├── docker-compose.yml
├── docker-compose.dev.yml       # dev overrides (hot reload, local ports)
├── Makefile
└── .env.example
```

## Backend Package Structure

Each module under `internal/` follows the same layout:

```
internal/<module>/
├── handler.go      # HTTP handlers, request parsing, response writing
├── service.go      # business logic, orchestrates across stores
├── store.go        # DB interface definition + SQL implementation
└── model.go        # domain types for this module
```

- Handlers know nothing about SQL
- Services know nothing about HTTP
- Stores know nothing about business rules

The structure is consistent regardless of module size. This makes AI-assisted development predictable — adding a field or endpoint always touches the same set of files in the same locations.

## Module Responsibilities

| Module | Owns | Notes |
|---|---|---|
| `auth` | Session management, auth middleware | Own module to support future user management |
| `cases` | Case lifecycle and status | Imports `people` |
| `people` | Person records, PersonRelationship | Imports nothing internal |
| `claimlines` | ClaimLine records and status | Imports `people` |
| `lifeevents` | LifeEvent records | Imports `people` |
| `documents` | Document records, FileAttachment, canonical file logic | Imports `lifeevents`, `storage` |
| `storage` | File upload/download, object storage abstraction | Imports nothing internal |
| `exports` | ZIP assembly, export downloads | Imports `cases`, `documents`, `storage` |

## Dependency Direction

```
auth        ← imported by platform (middleware)
cases       ← imports people
people      ← imports nothing internal
claimlines  ← imports people
lifeevents  ← imports people
documents   ← imports lifeevents, storage
storage     ← imports nothing internal
exports     ← imports cases, documents, storage
```

No cycles. `exports` is a leaf — nothing imports it.

## Key Constraints

- `storage` never imports `documents`. Storage knows about bytes and object keys only.
- `documents` imports `storage` via an interface, keeping storage testable and swappable.
- `platform/` is pure infrastructure — server setup, config, DB pool. No business logic.
- Frontend feature structure mirrors backend module structure. A feature change touches predictable files in both.

## Migration Convention

- Tool: `golang-migrate`
- Files: numbered SQL with `.up.sql` suffix, e.g. `001_initial_schema.up.sql`, `002_add_life_events.up.sql`
- No down migrations (`.down.sql` files are not used)
- `db/schema.sql` is kept as a current snapshot of the full schema for reference
