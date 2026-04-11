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

- [ ] Initial migration — full schema for all entities including DocumentStatus seed data
- [ ] Seed script with the Rossi → Martini case
- [ ] `GET /api/v1/cases` — list all cases
- [ ] `GET /api/v1/cases/:caseId` — case detail including ClaimLine summary and progress counts
- [ ] `GET /api/v1/cases/:caseId/claim-lines` — list ClaimLines for a case
- [ ] `GET /api/v1/cases/:caseId/people` — list People for a case
- [ ] `GET /api/v1/cases/:caseId/life-events` — list LifeEvents for a case, flagged if no documents
- [ ] `GET /api/v1/cases/:caseId/documents` — list Documents for a case with status and progress_bucket

### Frontend

- [ ] Case list page — shows all cases with title and status
- [ ] Case overview page — ClaimLine status summary, document progress breakdown by bucket (not_started / in_progress / complete), LifeEvents with no associated documents flagged
- [ ] People page — flat list of people in the case
- [ ] Documents page — list of documents with status and verification state

### Infrastructure

- [ ] Docker Compose with PostgreSQL and MinIO
- [ ] Go server with `platform/` wired, `golang-migrate` running on startup
- [ ] Vite dev server proxying API requests to Go backend
- [ ] `.env.example` with all required variables documented
- [ ] `Makefile` with at minimum: `dev`, `migrate`, `seed` targets
