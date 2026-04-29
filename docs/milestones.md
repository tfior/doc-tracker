# Milestones

## Milestone 1 — "A case exists and I can see what's in it"

Read-only, no auth, no file upload. Goal is to validate the full stack and the domain model against realistic data before any write paths are built.

### Acceptance Criteria

- The full dev stack runs with a single command (`docker compose up` or `make dev`)
- The database is migrated and seeded automatically on startup
- The case overview page displays meaningful progress information drawn from real data
- All listed frontend pages are navigable without errors
- No write operations, no auth, no file upload

### Seed Data

**Case:** Rossi → Martini  
**Scenario:** Sofia Martini applying for Italian citizenship by descent through her paternal line. LIRA is Giuseppe Rossi, who emigrated from Italy and naturalized in the US *after* Antonio (his son) was born — preserving the citizenship transmission.

**Lineage:**
```
Giuseppe Rossi (LIRA, born Italy, naturalized USA after Antonio's birth)
└── Antonio Rossi (born USA, Italian citizenship by descent)
    └── Carlo Rossi (born USA, Italian citizenship by descent)
        └── Sofia Martini née Rossi (applicant, born USA)
```

**People and LifeEvents:**

| Person | Events |
|---|---|
| Giuseppe Rossi | Birth (Italy), Marriage (to Maria Ferretti), Death |
| Antonio Rossi | Birth (USA), Marriage (to Maria Conti), Death |
| Carlo Rossi | Birth (USA), Marriage (to Linda Walsh), Death |
| Sofia Martini née Rossi | Birth (USA), Marriage (to Marco Martini) |

**Documents and statuses:**

| Person | Document | Status | Notes |
|---|---|---|---|
| Giuseppe | Italian birth certificate | verified | recorded_given_name: "Giuseppe", recorded_surname: "Rossi" |
| Giuseppe | Marriage certificate | verified | |
| Giuseppe | US naturalization certificate | collected, unverified | Key document — date must be after Antonio's birth |
| Giuseppe | Death certificate | pending | Not yet obtained |
| Antonio | US birth certificate | verified | Proves birth before Giuseppe's naturalization |
| Antonio | Marriage certificate | verified | |
| Antonio | Death certificate | collected, unverified | |
| Carlo | US birth certificate | verified | |
| Carlo | Marriage certificate | collected, unverified | |
| Carlo | Death certificate | pending | |
| Sofia | US birth certificate | verified | |
| Sofia | Marriage certificate | pending | No documents yet — flags as LifeEvent without documents |

**ClaimLine:** One, status `confirmed`, root person Giuseppe Rossi.

**Attachment notes:**
- Giuseppe's naturalization certificate has two FileAttachments: an original scan and an amended version (amendment added middle initial "L." to his name). The amendment is canonical; the original is superseded.
- All other documents with status `collected` or `verified` have one canonical FileAttachment.
- Pending documents have no FileAttachments.

---

### Backend

- [x] Initial migration — full schema for all entities including DocumentStatus seed data
- [x] Seed script with the Rossi → Martini case
- [x] `GET /api/v1/cases` — list all cases
- [x] `GET /api/v1/cases/:caseId` — case detail including ClaimLine summary and progress counts
- [x] `GET /api/v1/cases/:caseId/claim-lines` — list ClaimLines for a case
- [x] `GET /api/v1/cases/:caseId/people` — list People for a case
- [x] `GET /api/v1/cases/:caseId/life-events` — list LifeEvents for a case, flagged if no documents
- [x] `GET /api/v1/cases/:caseId/documents` — list Documents for a case with status and progress_bucket

### Frontend

- [x] Case list page — shows all cases with title and status
- [x] Case overview page — ClaimLine status summary, document progress breakdown by bucket (not_started / in_progress / complete), LifeEvents with no associated documents flagged
- [x] People page — flat list of people in the case
- [x] Documents page — list of documents with status and verification state

### Infrastructure

- [x] Docker Compose with PostgreSQL and MinIO
- [x] Go server with `platform/` wired, `golang-migrate` running on startup
- [x] Vite dev server proxying API requests to Go backend
- [x] `.env.example` with all required variables documented
- [x] `Makefile` with at minimum: `dev`, `migrate`, `seed` targets

---

## Milestone 2 — "I can create and manage a case"

Authentication and full write operations. After this milestone the app is a functional case management tool — a user can log in and build a case from scratch.

### Acceptance Criteria

- All routes (API and frontend) are protected; unauthenticated requests redirect to login
- A user can log in and log out
- A user can create, edit, and archive a case
- A user can add, edit, and remove people from a case
- A user can define parent-child relationships between people
- A user can add, edit, and remove life events
- A user can add, edit, and remove documents (metadata only — no file upload yet)
- A user can manually transition a document's status
- A user can create and update claim lines
- Deleted entities go to a trash view; trashed entities are frozen until restored or permanently deleted
- A user can restore a trashed entity or permanently delete it immediately
- A user can reassign a LifeEvent to a different Person within the same Case
- A user can reassign a Document to a different LifeEvent within the same Case (including cross-person)
- No file upload, no ZIP export

### Backend

- [x] Migration for `users` table
- [x] Migration for `activity_logs` table
- [x] `users` module: User model, store, service; bcrypt password hashing
- [x] `auth` module: login, logout, session middleware; authentication against `users` table
- [x] Auth middleware applied to all `/api/v1` routes
- [ ] Migration adding `deleted_at` to cases, people, life_events, documents, claim_lines; `ON DELETE CASCADE` on all entity FK constraints
- [ ] `POST /api/v1/cases`, `PATCH /api/v1/cases/:caseId`, `DELETE /api/v1/cases/:caseId` (soft-delete)
- [ ] `POST /api/v1/cases/:caseId/people`, `PATCH /api/v1/cases/:caseId/people/:personId`, `DELETE /api/v1/cases/:caseId/people/:personId` (soft-delete)
- [ ] `POST /api/v1/cases/:caseId/people/:personId/relationships`, `DELETE /api/v1/cases/:caseId/people/:personId/relationships/:parentId` (hard-delete)
- [ ] `POST /api/v1/cases/:caseId/life-events`, `PATCH /api/v1/cases/:caseId/life-events/:eventId`, `DELETE /api/v1/cases/:caseId/life-events/:eventId` (soft-delete)
- [ ] `PATCH /api/v1/cases/:caseId/life-events/:eventId/person` — reassign LifeEvent to a different Person within the same Case
- [ ] `POST /api/v1/cases/:caseId/documents`, `PATCH /api/v1/cases/:caseId/documents/:docId`, `DELETE /api/v1/cases/:caseId/documents/:docId` (soft-delete)
- [ ] `PATCH /api/v1/cases/:caseId/documents/:docId/status` — manual status transition
- [ ] `PATCH /api/v1/cases/:caseId/documents/:docId/parent` — reassign Document to a different LifeEvent/Person within the same Case
- [ ] `POST /api/v1/cases/:caseId/claim-lines`, `PATCH /api/v1/cases/:caseId/claim-lines/:lineId`, `DELETE /api/v1/cases/:caseId/claim-lines/:lineId` (soft-delete)
- [ ] Trash endpoints: list trashed entities, restore, permanent delete
- [ ] Activity log insertion in all write handlers (create, update, delete, restore, reassign)

### Frontend

- [x] Login page and logout action
- [x] Auth-aware routing — redirect to login if no active session
- [ ] Create and edit case forms
- [ ] Add, edit, and remove person forms
- [ ] Parent-child relationship UI (Parents field: up to 2; Children field: unlimited; same-case scope)
- [ ] Add, edit, and remove life event forms
- [ ] Reassign life event to a different person
- [ ] Add, edit, and remove document forms
- [ ] Reassign document to a different life event / person
- [ ] Document status transition UI
- [ ] Claim line create and status management UI
- [ ] Trash view — list trashed entities with restore and permanent delete actions

### Infrastructure

- [x] Sessions stored server-side in memory (database-backed sessions deferred)
- [x] `make create-user` target — prompts for email, first name, last name, password; inserts a new user record; separate from seed data
- [x] GitHub Actions CI — runs `go test ./...` against a postgres service on push to main and on all pull requests
