# Milestones

The design docs (`domain-model.md`, `architecture.md`, `api-conventions.md`) describe the agreed design. This file is the only place that says what has been built. A task is ticked when its code has landed, with its tests.

| Milestone | Outcome | Status |
|---|---|---|
| 1 | A case exists and can be viewed | Done |
| 2 | A case can be created and managed | Done |
| 3 | Integrity: the data can be trusted | Next |
| 4 | Dates and life events | |
| 5 | Readiness | |
| 6 | Cases and applications | |
| 7 | Deployable — **MVP complete** | |
| 8 | Files | After the MVP |
| 9 | Exports | After the MVP |

**Working rules for every milestone**

- Test scenarios are proposed and agreed before tests are written, and tests are written before the implementation.
- Every domain rule a milestone touches gets a test that would fail if the rule were not enforced.
- When a branch touches a file, it also clears that file's entries in the Small Fixes checklist at the end of this document.
- A migration never discards data.
- From Milestone 7 onward, a branch that adds a migration runs it against a copy of a realistic database before merging, not only an empty one.
- A pull request is ready when: all tests pass; the milestone's tasks it completes are ticked; the design docs agree with the code; and the Small Fixes entries for every file it touched are cleared.

---

## Milestone 1 — "A case exists and I can see what's in it"

Read-only, no auth, no file upload. The goal was to validate the full stack and the domain model against realistic data before any write paths were built.

### Acceptance Criteria

- The full dev stack runs with a single command (`docker compose up` or `make dev`)
- The database is migrated and seeded automatically on startup
- The case overview page displays meaningful progress information drawn from real data
- All listed frontend pages are navigable without errors
- No write operations, no auth, no file upload

### Seed Data

One case, Rossi → Martini: Sofia Martini claiming through her great-grandfather Giuseppe Rossi, with four generations of people, their life events, and their documents in a mix of states. The seed cases are redefined in Milestone 4.

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
- A user can create and update claim lines (root person + status + notes; explicit path through the lineage graph deferred post-MVP)
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
- [x] Migration adding `deleted_at` to cases, people, life_events, documents, claim_lines; `ON DELETE CASCADE` on all entity FK constraints
- [x] `POST /api/v1/cases`, `PATCH /api/v1/cases/:caseId`, `DELETE /api/v1/cases/:caseId` (soft-delete)
- [x] `POST /api/v1/cases/:caseId/people`, `PATCH /api/v1/cases/:caseId/people/:personId`, `DELETE /api/v1/cases/:caseId/people/:personId` (soft-delete)
- [x] `POST /api/v1/cases/:caseId/people/:personId/relationships`, `DELETE /api/v1/cases/:caseId/people/:personId/relationships/:parentId` (hard-delete)
- [x] `POST /api/v1/cases/:caseId/life-events`, `PATCH /api/v1/cases/:caseId/life-events/:eventId`, `DELETE /api/v1/cases/:caseId/life-events/:eventId` (soft-delete)
- [x] `PATCH /api/v1/cases/:caseId/life-events/:eventId/person` — reassign LifeEvent to a different Person within the same Case
- [x] `POST /api/v1/cases/:caseId/documents`, `PATCH /api/v1/cases/:caseId/documents/:docId`, `DELETE /api/v1/cases/:caseId/documents/:docId` (soft-delete)
- [x] `PATCH /api/v1/cases/:caseId/documents/:docId/parent` — reassign Document to a different LifeEvent/Person within the same Case
- [x] `POST /api/v1/cases/:caseId/claim-lines`, `PATCH /api/v1/cases/:caseId/claim-lines/:lineId`, `DELETE /api/v1/cases/:caseId/claim-lines/:lineId` (soft-delete)
- [x] Trash endpoints: list trashed entities, restore, permanent delete
- [x] Activity log insertion in all write handlers (create, update, delete, restore, reassign)
- [x] Migration: replace `status_id` on documents with four phase status columns (`official_copy_status_id`, `amendment_status_id`, `apostille_status_id`, `translation_status_id`); add `phase` column to `document_statuses`; reseed with new system statuses
- [x] Update Document model, store, and service for four-phase status fields
- [x] `GET /api/v1/document-statuses` — list all document statuses, filterable by `?phase=`
- [x] `PATCH /api/v1/cases/:caseId/documents/:docId/status` — revised to accept `{phase, status_id}` for per-phase transitions

### Frontend

- [x] Login page and logout action
- [x] Auth-aware routing — redirect to login if no active session
- [x] Create and edit case forms
- [x] Add, edit, and remove person forms
- [x] Parent-child relationship UI (Parents field: up to 2; Children field: unlimited; same-case scope)
- [x] Add, edit, and remove life event forms
- [x] Reassign life event to a different person
- [x] Add, edit, and remove document forms
- [x] Reassign document to a different life event / person
- [x] Document status transition UI (inline in Documents table + within edit modal)
- [x] Claim line create and status management UI
- [x] Trash view — list trashed entities with restore and permanent delete actions (per-case trash for people, life events, documents, claim lines; global trash in sidebar for deleted cases)

### Infrastructure

- [x] Sessions stored server-side in memory (database-backed sessions deferred)
- [x] `make create-user` target — prompts for email, first name, last name, password; inserts a new user record; separate from seed data
- [x] GitHub Actions CI — runs `go test ./...` against a postgres service on push to main and on all pull requests

---

## Milestone 3 — Integrity: "The data can be trusted"

Makes every write all-or-nothing, gives each rule a single owner, and fixes the defects found in the October 2026 review. No new user-facing features. This milestone reworks every service, store, and handler, so the later milestones build on a sound base.

### Acceptance Criteria

- Every write, together with its activity log entry, either happens completely or not at all
- A save from any form is a single request
- Nothing in the trash, or beneath something in the trash, appears in a list or can be changed
- Invalid input never produces a server error; the user sees a message beside the field at fault
- Two users editing the same record cannot silently overwrite each other
- Each defect listed below has a test that fails without its fix

### Backend

Structure
- [x] Transaction helper in `platform`; stores run on the transaction their service opened
- [x] Services own each unit of work: the change and its activity log entry in one transaction
- [x] Module boundaries: modules declare the interfaces they need, wired in `internal/app`; each module is the only writer of its tables
- [x] Each owning module answers "is this active?" for its entities: exists, in the case, not trashed, not beneath a trashed parent
- [x] `trash` becomes a coordinator; owning modules restore and permanently delete their own records
- [x] Confirm gone: the per-handler logging helper, the per-module nullable field type, the repeated person-in-case check

Defects from the review
- [x] Circular parent links are refused
- [x] Reassigning a life event moves its documents with it
- [x] A document's person always matches its life event's person, on create and on reassignment
- [x] Lists exclude everything beneath a trashed parent; writes beneath a trashed parent or in a trashed case are refused
- [x] The claim line summary excludes trashed claim lines
- [x] The two-parent limit counts only active parents
- [x] Permanently deleting a life event deletes its documents (migration: change the foreign key to CASCADE)
- [x] An active check locks the row it checks until the write commits, so a parent trashed at the same moment cannot receive a new child

Activity log
- [ ] `changes` holds the old and new value of each field that actually changed
- [ ] Human-readable entity names everywhere, including on delete
- [ ] `restored` and `permanently_deleted` actions; restore and permanent delete are logged
- [ ] CaseDeletionRecord written when a case is permanently deleted
- [ ] Tests assert on activity log rows

API
- [ ] `version` on Case, Person, LifeEvent, Document, ClaimLine; a stale write is refused with `conflict`
- [ ] One `PATCH` per entity accepting everything its form can change; remove the status, parent, person, and relationship endpoints
- [ ] Lists inside a case are returned in full
- [ ] Error responses carry field-level errors with codes; every invalid field is reported at once
- [ ] Malformed IDs and invalid values return `invalid_input`, not a server error
- [ ] Unknown request fields are rejected; names are trimmed and must not be blank
- [ ] Every server error is logged with its cause
- [ ] Deleting a person or life event reports what goes with it, for the confirmation dialog
- [x] Handler tests run behind the auth middleware

### Frontend

- [ ] Each form's Save sends one request, including the record's version
- [ ] Field errors are shown beside their fields; a server message is never replaced by a generic one
- [ ] A conflict tells the user the record changed and offers to reload it
- [ ] An expired session on any request sends the user to the login screen
- [ ] Every action reports failure, including the inline status dropdowns and all delete buttons
- [ ] Delete confirmations state what will go with the item
- [ ] Pages touched are split into one component file per form, modal, and section

### Infrastructure

- [ ] `make schema` generates `db/schema.sql` from a freshly migrated database; CI fails if the committed file is stale
- [x] CI runs `gofmt`, `go vet`, and `govulncheck`

---

## Milestone 4 — Dates and life events

### Acceptance Criteria

- A date can be entered as precisely or as roughly as it is known, and narrowed later
- A person's birth and death are held in one place
- A marriage between two people in a case is one event with one set of documents
- No screen or document says "LIRA" or "root person"

### Backend

- [ ] Partial date and qualified date types, with validation and the earliest and latest day each represents
- [ ] Migration: life event dates become qualified dates and document dates become partial dates, preserving existing values
- [ ] `recorded_date` renamed to `registration_date`
- [ ] Person birth and death fields removed; values migrated into birth and death life events
- [ ] At most one birth event and one death event per person; saving a person writes them in the same transaction
- [ ] `divorce_date` and `divorce_place` on marriage events
- [ ] `spouse_person_id` on marriage events; a person's life events include marriages where they are the linked spouse
- [ ] A shared marriage survives the deletion of one spouse
- [ ] Spouse and divorce fields are refused on events that are not marriages
- [ ] Seed data: the Rossi case with its line marked ineligible, and a second case through a grandparent marked eligible; the seed goes through the services, not raw SQL

### Frontend

- [ ] A shared date input in `components/`: qualifier plus one or two partial dates
- [ ] The person form edits birth and death through their life events
- [ ] The marriage form can link the spouse to a person in the case, and has a "Marriage ended in divorce" section
- [ ] Wording: "Root Ancestor", "Registration date", and the "As recorded in document" heading
- [ ] Qualified dates are displayed in words, such as "about 1890"

---

## Milestone 5 — Readiness

After this milestone the app answers "what is still outstanding on this case?".

### Acceptance Criteria

- Every document shows whether it is ready and, if not, why
- A case shows how many of its documents are ready, and lists the rest by what they are waiting on
- A request's date and reference number are recorded, and the time it has been waiting is visible
- Everything a document records about anyone can be compared side by side for a person

### Backend

- [ ] `is_verified` becomes `details_checked`, recording who and when; setting it on create works; its timestamp no longer changes on unrelated saves
- [ ] Readiness on every document: status, the stage it is waiting on, and reasons
- [ ] Progress counts on every case, in the list and the detail
- [ ] A since date on each phase, set automatically and correctable; a reference on each phase
- [ ] The `for_submission` flag
- [ ] Document types: non-naturalization certificate, divorce record, court order
- [ ] `recorded_age` on documents, returned with the birth window it implies
- [ ] DocumentNamedPerson, saved and logged with its document
- [ ] The recorded-details comparison for a person, birth certificate first as the reference
- [ ] The age of a document's current copy

### Frontend

- [ ] Documents table: readiness with reasons, and time in the current status; the phase labelled "Apostille / Legalization"
- [ ] Case overview: progress summary and the outstanding list by stage, sortable by time waiting; every row names its person
- [ ] Document form: "Details checked against scan" with who and when; since date and reference per phase; "for submission"; "Other people named in this document"
- [ ] Person screen: the comparison table

---

## Milestone 6 — Cases and applications

### Acceptance Criteria

- A case can be put on hold, completed, and archived; an archived case cannot be changed
- A case pursues one root ancestor, and each applicant one eligible claim line to it
- Each claim line records the exact path from root ancestor to applicant
- Applicants are grouped into applications, each showing the documents it needs and whether it is ready for submission
- The case list can be filtered by status and searched by title

### Backend

- [ ] Case status becomes active, on hold, or complete; `archived_at` with archive and unarchive actions; notes
- [ ] An archived case is read-only, enforced where "is this active?" is answered
- [ ] Case list: status filter, archived filter, title search
- [ ] `pursued_root_person_id` replaces `primary_root_person_id`, with its rules
- [ ] Claim lines gain an applicant and a stored path, found from the family links; no two with the same path, and for now one per applicant and root ancestor
- [ ] The pursued flag: one per applicant, eligible only, to the case's pursued root ancestor; status locked while pursued
- [ ] A parent link used by a path cannot be removed; permanently deleting a person deletes the claim lines whose path includes them
- [ ] `applications` module: applications and their applicants
- [ ] An application's documents, from each applicant's pursued claim line, each counted once
- [ ] An applicant without a pursued claim line is reported
- [ ] Application status, with the number of documents ready out of the total
- [ ] Copies required on every document

### Frontend

- [ ] Case form: status, notes, archive and unarchive
- [ ] Case list: status filter, archived view, title search, progress beside each case
- [ ] Claim Lines screen grouped by applicant: choose the case's pursued root ancestor, mark each applicant's pursued line with lines to other ancestors dimmed, and choose a path when more than one exists
- [ ] Applications screen: create and edit applications; each shows its documents in packet order with their readiness and the age of each copy
- [ ] Deleting an application warns that it is permanent

---

## Milestone 7 — Deployable

**The MVP is complete when this milestone is done and the Small Fixes checklist is empty.**

### Acceptance Criteria

- The application deploys as one image behind HTTPS, following a reference deployment in the repo
- A first user can be created, and a password reset, on a deployed copy
- A restart or deploy does not log users out
- A backup has been restored successfully, following written instructions

### Running it

- [ ] The backend serves the built frontend; one image
- [ ] Reference deployment: production Compose file and Caddy configuration
- [ ] Create-user and password-reset commands inside the image
- [ ] Request logging and a clean shutdown
- [ ] The container runs as a non-root user
- [ ] End-to-end tests for the critical paths: logging in, creating a case with people, life events and documents, changing a document's status, and seeing it become ready

### Security

- [ ] Sessions stored in the database
- [ ] HTTPS-only session cookie in production
- [ ] Login throttling
- [ ] Request size limits and server timeouts
- [ ] Email addresses compared without regard to case
- [ ] Passwords are never trimmed; creating a user asks for the password twice and enforces a minimum length
- [ ] Database connections can use TLS; passwords with special characters work
- [ ] The server refuses to start with placeholder secrets; the unused `SESSION_SECRET` is removed

### Data safety

- [ ] The 30-day trash purge, writing a CaseDeletionRecord for each purged case
- [ ] Backup instructions, and a restore that has been tested
- [ ] The backup-before-upgrade rule written into the deployment instructions

---

## Milestone 8 — Files (after the MVP)

Scans are attached to documents and can be opened or downloaded one at a time.

- [ ] Choose a local S3-compatible store for development and add it to Docker Compose
- [ ] The `storage` module, behind an interface
- [ ] FileAttachment reshaped: kind, current marker, uploaded by
- [ ] Upload: PDF only for submission files, PDF or image for supporting files, checked by content; a configurable size limit
- [ ] The versioned submission file: the newest upload is current, earlier versions are kept
- [ ] Supporting files
- [ ] Open a file in the browser, or download it, from a case-scoped URL
- [ ] The upload option "This scan changes the recorded details", which clears the details check; the check records which version it was made against
- [ ] Stored files are removed when their document is permanently deleted
- [ ] Encryption at rest shown in the reference deployment

### Open Decisions

- **Local object storage for development.** MinIO was removed from Docker Compose because its image is no longer published and the project is no longer maintained. An S3-compatible replacement must be chosen before the `storage` module is built. The `STORAGE_*` variables in `.env.example` are retained for it.

---

## Milestone 9 — Exports (after the MVP)

- [ ] Application Files export for one application: a ZIP of the current submission file of every ready document
- [ ] Generated file names and packet order
- [ ] The manifest inside the ZIP: every document, included or left out, and why
- [ ] All Files export for a case

---

## Small Fixes

Minor issues found in the October 2026 review. Each is cleared when its file is next touched, and **all must be ticked before the MVP is complete**, whether or not their files were touched along the way. The milestone in brackets is where each is expected to fall.

- [x] Go files formatted to `gofmt` standard (3)
- [x] A document cannot be restored from the trash while its person or life event is still in it (3)
- [x] Two simultaneous requests cannot give a person a third parent (3)
- [x] Lists for a case that does not exist return not found, not an empty list (3)
- [ ] Failed requests are not retried when the answer is definite, such as not found (3)
- [ ] Logging out clears cached data, so the next user on the same browser never sees it (3)
- [ ] An unknown URL shows a "page not found" screen
- [ ] After logging in, the user lands on the page they were heading for
- [ ] A crash in one component shows an error message, not a blank screen
- [x] Lists have a stable order when dates are equal or missing
- [ ] Spouse details cannot be saved on events that are not marriages (4)
- [ ] The overview's list of life events without documents names each event's person (5)
- [ ] Login takes the same time for an unknown email as for a wrong password (7)
- [ ] Creating a user asks for the password twice and enforces a minimum length (7)
