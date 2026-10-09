# doc-tracker

A case management tool for citizenship-by-descent document collection. It is built for professionals who gather vital records on behalf of applicants: it tracks the people in a lineage, the life events that need proving, and the status of every document through collection, amendment, apostille and translation.

## Status

The app is a working case manager but not yet feature-complete.

- **Done:** login, cases, people and parent-child relationships, life events, documents with per-phase statuses, claim lines, and a trash with restore.
- **Not yet built:** file upload and ZIP export of a case's files.

See [docs/milestones.md](docs/milestones.md) for the detailed checklist.

## Stack

| Layer | Technology |
|---|---|
| Frontend | React, TypeScript, Vite, Mantine, TanStack Query, React Router |
| Backend | Go (modular monolith, standard library HTTP server) |
| Database | PostgreSQL 18, plain SQL migrations via golang-migrate |
| Dev environment | Docker Compose, Make |

## Running locally

### Prerequisites

- Docker with the Compose plugin
- Go 1.27 and Node 24. Exact versions are pinned in [.tool-versions](.tool-versions); with [asdf](https://asdf-vm.com), `asdf install` sets up both.
- Make

### First-time setup

```bash
cp .env.example .env
cd frontend && npm ci && cd ..
```

### Start the app

Each of the first three commands keeps running, so use a separate terminal for each.

```bash
make dev          # PostgreSQL in Docker
make backend      # API on http://localhost:8080; applies migrations on startup
make frontend     # UI on http://localhost:5173
```

Then, once the backend is up:

```bash
make seed         # loads a sample case (safe to run more than once)
make create-user  # prompts for email, name and password
```

Open http://localhost:5173 and log in with the user you created.

## Tests

```bash
make test                      # backend integration tests; needs `make dev` running
cd frontend && npm run lint    # frontend lint
cd frontend && npm run build   # frontend type-check and build
```

The backend tests run against a real PostgreSQL database (`doctracker_test`), which the test helper creates on first use. CI runs all three on every pull request.

## Documentation

| Document | Covers |
|---|---|
| [docs/domain-model.md](docs/domain-model.md) | Entities, fields and domain rules |
| [docs/architecture.md](docs/architecture.md) | Stack choices, module boundaries, repo structure |
| [docs/api-conventions.md](docs/api-conventions.md) | HTTP API conventions |
| [docs/milestones.md](docs/milestones.md) | Milestones and task checklists |

## Deployment

The app is designed to be self-hostable. All infrastructure settings come from environment variables (see [.env.example](.env.example)), and no hosting provider is assumed.
