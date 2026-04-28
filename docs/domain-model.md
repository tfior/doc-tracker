# Domain Model

## Entities

### User
An authenticated user of the application. All users have shared access to all cases — no per-user permissions or workspaces for MVP. Users are created via a CLI command, not through a registration flow.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| email | text | unique; used as login identifier |
| first_name | text | |
| last_name | text | |
| password_hash | text | bcrypt |
| created_at | timestamp | |
| updated_at | timestamp | |

---

### ActivityLog
An append-only record of a write action performed by a user. Captured on every create, update, and delete across all case-scoped entities. Display (activity feed UI) is deferred post-MVP; data capture is required from Milestone 2 onward.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| user_id | uuid | FK → User |
| action | enum | created, updated, deleted |
| entity_type | enum | case, person, person_relationship, life_event, document, file_attachment, claim_line |
| entity_id | uuid | ID of the affected entity |
| entity_name | text | Human-readable label at the time of the action (e.g. "Birth life event for Giuseppe Rossi", "Italian Birth Certificate") |
| changes | jsonb? | Field-level diffs for updates: `[{"field": "first_name", "from": "Mario", "to": "Luigi"}]`; null for created and deleted |
| created_at | timestamp | |

---

### Case
The top-level container for a document collection effort. One case may have multiple applicants and multiple candidate lineage lines.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| title | text | |
| status | enum | active, archived, complete |
| primary_root_person_id | uuid? | FK → Person; nullable until LIRA is confirmed |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### ClaimLine
One candidate lineage path through the case's person graph. Always present, even for simple single-LIRA cases (status: eligible). Supports mid-case pivots and multi-line eligibility research without structural rebuilds.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| root_person_id | uuid | FK → Person; immutable after creation |
| status | enum | not_yet_researched, researching, paused, ineligible, eligible |
| notes | text? | |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### Person
A person in the lineage graph, scoped per case for MVP. `first_name` and `last_name` are working identity — not verified facts. Verified name data lives on Document. Persons cannot be reassigned across Cases.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| first_name | text | |
| last_name | text | |
| birth_date | date? | |
| birth_place | text? | |
| death_date | date? | |
| notes | text? | |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

**Deferred:** `canonical_person_id` (nullable FK → Person) for cross-case linking. Add when needed.

---

### PersonRelationship
Parent-child relationships only. Supports multiple parents per person to handle branching lineage lines. Marriages and other relationships are captured via LifeEvent, not here.

PersonRelationship is a bare join table with no soft-delete. Deletions are hard-deletes; the user re-adds the relationship if removed by mistake. The UI exposes a **Parents** field per Person (up to 2 selectable People within the same Case) and a **Children** field (unlimited). Circular reference validation (ancestor/descendant cycles) is enforced in the application layer.

| Field | Type | Notes |
|---|---|---|
| person_id | uuid | FK → Person |
| parent_id | uuid | FK → Person |
| case_id | uuid | FK → Case |

---

### LifeEvent
A significant event in a person's life that may require one or more documents to prove. The in-line subject (`person_id`) is the lineage-relevant person. Spouse data is stored as flat fields — spouses are not Person records in the graph.

A LifeEvent may be reassigned to a different Person within the same Case. Cross-case reassignment is not permitted.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| person_id | uuid | FK → Person (in-line subject); reassignable within same Case |
| event_type | enum | birth, marriage, death, naturalization, immigration, other |
| event_date | date? | |
| event_place | text? | |
| spouse_name | text? | marriage events only |
| spouse_birth_date | date? | marriage events only |
| spouse_birth_place | text? | marriage events only |
| notes | text? | |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### DocumentStatus
Defines the status options available for a Document. System statuses are seeded at migration time and cannot be edited or deleted. User-defined statuses are global (shared across all cases) and always map to the `in_progress` progress bucket. Progress logic always operates on `progress_bucket`, never on user-defined labels.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| label | text | display name |
| is_system | boolean | true for seeded system statuses |
| system_key | enum? | pending, collected, verified, unobtainable — null for user-defined |
| progress_bucket | enum | not_started, in_progress, complete |
| created_at | timestamp | |

**System statuses (seeded at migration):**

| system_key | label | progress_bucket |
|---|---|---|
| `pending` | Pending | not_started |
| `collected` | Collected | in_progress |
| `verified` | Verified | complete |
| `unobtainable` | Unobtainable | complete |

**Deferred:** Admin UI for managing user-defined statuses. The table is ready for it; the UI is post-MVP.

---

### Document
A specific official record belonging to a Person, optionally grouped under a LifeEvent. `recorded_*` fields capture data exactly as it appears on the current canonical file, in the original document's original language only. Translations and apostilles do not contribute to recorded metadata.

`is_verified` flips to false when a new canonical FileAttachment of type `amendment` is uploaded, forcing re-verification. Apostille and translation uploads do not affect `is_verified`.

The system automatically transitions `status` to `collected` when a canonical FileAttachment is first uploaded. All other status transitions are manual.

A Document may be reassigned to a different LifeEvent within the same Case, including one belonging to a different Person. When reassigned cross-person, `person_id` and `life_event_id` are updated atomically. Setting `life_event_id` to null ungrouped the Document under the Person with no event association.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| person_id | uuid | FK → Person; updated atomically when reassigned to an event under a different Person |
| life_event_id | uuid? | FK → LifeEvent; null = ungrouped; reassignable within same Case |
| status_id | uuid | FK → DocumentStatus; default: pending system status |
| document_type | enum | birth_certificate, marriage_certificate, naturalization, death_certificate, other |
| title | text | |
| issuing_authority | text? | |
| issue_date | date? | |
| recorded_date | date? | date the event was recorded (may differ from event date) |
| recorded_given_name | text? | as it appears on the canonical file, original language |
| recorded_surname | text? | as it appears on the canonical file, original language |
| recorded_birth_date | date? | as it appears on the canonical file |
| recorded_birth_place | text? | as it appears on the canonical file |
| is_verified | boolean | default false; flips false on amendment upload |
| verified_at | timestamp? | |
| notes | text? | AKA variants, discrepancies, amendment history narrative |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### FileAttachment
A physical file attached to a Document. One attachment is marked canonical at any given time. When a new canonical file is uploaded, the previous canonical attachment is superseded (superseded_at is set) but retained for audit purposes.

`attachment_type` drives re-verification behavior: amendment flips `Document.is_verified` to false; apostille and translation do not.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| document_id | uuid | FK → Document |
| storage_key | text | object storage path |
| filename | text | |
| content_type | text | |
| size_bytes | integer | |
| is_canonical | boolean | |
| attachment_type | enum | original, apostille, translation, amendment |
| superseded_at | timestamp? | set when no longer canonical |
| notes | text? | |
| uploaded_at | timestamp | |

---

## Domain Rules

1. `recorded_*` fields always reflect the original document in its original language. Never populate from translations or apostilles.
2. Uploading a new canonical file of type `amendment` flips `Document.is_verified` to false. Apostille and translation uploads do not.
3. Superseded FileAttachments are retained for audit purposes, never deleted.
4. Spouse data lives on the marriage LifeEvent as flat fields. Spouses are not Person records.
5. PersonRelationship is parent-child only.
6. ClaimLine is always present. Single-LIRA cases have one ClaimLine with status `confirmed`.
7. Person records are scoped per case for MVP.
8. `Case.primary_root_person_id` reflects the confirmed LIRA. It may be null during eligibility research.
9. Progress logic always operates on `DocumentStatus.progress_bucket`, never on user-defined status labels.
10. The system automatically sets Document status to `collected` when a canonical FileAttachment is first uploaded. All other status transitions are manual.
11. User-defined DocumentStatus entries are global (shared across all cases) and always assigned `progress_bucket = in_progress`.
12. System DocumentStatus entries cannot be edited or deleted by users.
13. Files are uploaded directly to object storage via the backend using multipart form upload. Presigned URLs are not used for MVP.
14. Object storage buckets must have server-side encryption (SSE) enabled. This is a deployment requirement, not enforced in application code.
15. ActivityLog entries are append-only. They are never updated or deleted.
16. `ActivityLog.entity_name` is captured at the time of the action and is not updated if the entity is later renamed or deleted.
17. `ActivityLog.changes` stores field-level diffs for `updated` actions only; null for `created` and `deleted`.
18. Activity log entries are written for all write operations on case-scoped entities (cases, people, person relationships, life events, documents, file attachments, claim lines).

### Soft-delete and Trash

19. Case, Person, LifeEvent, Document, and ClaimLine have a `deleted_at` column. A null value means active; a non-null value means trashed. PersonRelationship and FileAttachment have no soft-delete.
20. All read queries filter `WHERE deleted_at IS NULL`. Trashed entities are excluded from normal results and only appear in the trash view.
21. Only directly-deleted entities appear in the trash. Children of a trashed entity are hidden implicitly — their own `deleted_at` remains null and they do not appear in the trash independently.
22. A trashed entity and all its implicitly-hidden children are frozen: no edits, no reassignment, no new children may be added until the parent is restored.
23. Restoring a trashed entity automatically restores all implicitly-hidden children (they were never independently deleted).
24. Trashed entities are permanently hard-deleted after 30 days. All descendant entities are permanently deleted via DB-level `ON DELETE CASCADE` foreign keys.
25. If a LifeEvent is permanently deleted, its Documents are also permanently deleted (CASCADE). If the user wants a Document to survive the deletion of its LifeEvent, they must reassign the Document to another event (or ungroup it) before deleting the event.
26. If a user explicitly deletes a child entity while its parent is still active (e.g., deletes a Document while the Person and LifeEvent are active), that child appears in the trash independently and follows its own 30-day clock.
27. PersonRelationship uses hard-delete only. Deleted relationships are gone immediately with no trash period.

### Reassignment

28. A LifeEvent may be reassigned to a different Person within the same Case. Cross-case reassignment is not permitted.
29. A Document may be reassigned to a different LifeEvent within the same Case, including one belonging to a different Person. When the target event belongs to a different Person, `person_id` and `life_event_id` are updated atomically in a single transaction.
30. A Document's `life_event_id` may be set to null to ungroup it from any event, leaving it associated with its Person only.
31. Reassignment is not permitted if the entity being moved or its target parent is trashed.
32. Reassignment is logged as an `updated` action in ActivityLog with the old and new parent ID(s) captured in `changes`.
33. PersonRelationship graph validation is enforced in the application layer: a Person may have at most 2 parents, and no Person may be both a direct ancestor and a direct descendant of the same Person (no circular relationships).

---

## MoSCoW

### Must (MVP)
- User, Case, ClaimLine, Person, PersonRelationship, LifeEvent, Document, FileAttachment, DocumentStatus, ActivityLog
- Multi-user authentication (shared access, all users see all data, no per-user permissions); session-based
- Activity log data capture on all write operations (display deferred)
- File upload to S3-compatible storage via multipart form, server-side encryption at rest
- Case overview and progress tracking (ClaimLine status + document progress buckets + LifeEvents without documents)
- ZIP export of canonical files per case
- ZIP export of all files per case

### Could (post-MVP)
- Activity feed UI (per-case feed; per-entity feed)
- Checklist/Task entity for lightweight user-driven progress tracking
- Admin UI for managing user-defined DocumentStatus entries
- Async export job records with file manifest
- DocumentNameRecord — structured AKA / name variant per document
- NameDiscrepancy records with severity and resolution status
- DocumentAmendment structured tracking
- `canonical_person_id` on Person for cross-case linking
- Global person/document scope across cases
- Second case type for eligibility workflow
- Automatic discrepancy flagging across documents
- Presigned URL upload flow (if direct upload becomes a bottleneck)
- User registration flow

### Won't (MVP)
- RequirementSpec / CaseRequirement — adds complexity for little MVP benefit; checklist covers the need
- Per-user permissions, workspaces, or case assignments
- Public sharing portals
- Google Drive / GEDCOM / GRAMPS integration
- Full family tree builder
- Conditional requirement logic / rules engine
- OCR / AI extraction
- Workflow automation
- Microservices
