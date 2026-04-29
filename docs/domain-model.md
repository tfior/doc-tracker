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
Defines the status options available for each phase of a Document. Each status belongs to a specific phase (`official_copy`, `amendment`, `apostille`, `translation`) or to all phases (`any`). System statuses are seeded at migration time and cannot be edited or deleted. Progress logic operates on `progress_bucket`.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| label | text | display name |
| phase | enum | official_copy, amendment, apostille, translation, any — `any` statuses appear in all four phase pickers |
| is_system | boolean | true for seeded system statuses |
| system_key | text? | set only for the four phase-default statuses the backend references at document creation; null for all others |
| progress_bucket | enum | not_started, in_progress, complete |
| created_at | timestamp | |

**System statuses (seeded at migration):**

| label | phase | system_key | progress_bucket |
|---|---|---|---|
| Not Started | official_copy | official_copy_default | not_started |
| Not Started | apostille | apostille_default | not_started |
| Not Started | translation | translation_default | not_started |
| Unknown | amendment | amendment_default | not_started |
| Required — Not Started | amendment | — | not_started |
| Researching | official_copy | — | in_progress |
| Researching | amendment | — | in_progress |
| Requested | official_copy | — | in_progress |
| Requested | amendment | — | in_progress |
| Sent | apostille | — | in_progress |
| Sent | translation | — | in_progress |
| Ready for Review | any | — | in_progress |
| Complete | any | — | complete |
| Not Required | any | — | complete |

**Deferred:** Admin UI for managing user-defined statuses. Custom statuses will specify a phase and a progress_bucket. The table is ready for it; the UI is post-MVP.

---

### Document
A specific official record belonging to a Person, optionally grouped under a LifeEvent. `recorded_*` fields capture data exactly as it appears on the current canonical file, in the original document's original language only. Translations and apostilles do not contribute to recorded metadata.

`is_verified` is a manual boolean field that can be set independently of phase statuses. It is not automatically modified by any system action including file uploads.

A Document may be reassigned to a different LifeEvent within the same Case, including one belonging to a different Person. When reassigned cross-person, `person_id` and `life_event_id` are updated atomically. Setting `life_event_id` to null ungrouped the Document under the Person with no event association.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| person_id | uuid | FK → Person; updated atomically when reassigned to an event under a different Person |
| life_event_id | uuid? | FK → LifeEvent; null = ungrouped; reassignable within same Case |
| official_copy_status_id | uuid | FK → DocumentStatus (phase=official_copy or any); default: Not Started |
| amendment_status_id | uuid | FK → DocumentStatus (phase=amendment or any); default: Unknown |
| apostille_status_id | uuid | FK → DocumentStatus (phase=apostille or any); default: Not Started |
| translation_status_id | uuid | FK → DocumentStatus (phase=translation or any); default: Not Started |
| document_type | enum | birth_certificate, marriage_certificate, naturalization, death_certificate, other |
| title | text | |
| issuing_authority | text? | |
| issue_date | date? | |
| recorded_date | date? | date the event was recorded (may differ from event date) |
| recorded_given_name | text? | as it appears on the canonical file, original language |
| recorded_surname | text? | as it appears on the canonical file, original language |
| recorded_birth_date | date? | as it appears on the canonical file |
| recorded_birth_place | text? | as it appears on the canonical file |
| is_verified | boolean | default false; manual checkbox |
| verified_at | timestamp? | |
| notes | text? | AKA variants, discrepancies, amendment history narrative |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### FileAttachment
A physical file attached to a Document. One attachment is marked canonical at any given time. When a new canonical file is uploaded, the previous canonical attachment is superseded (superseded_at is set) but retained for audit purposes.

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
2. `Document.is_verified` is a manual boolean field. No system action modifies it automatically.
3. Superseded FileAttachments are retained for audit purposes, never deleted.
4. Spouse data lives on the marriage LifeEvent as flat fields. Spouses are not Person records.
5. PersonRelationship is parent-child only.
6. ClaimLine is always present. Single-LIRA cases have one ClaimLine with status `confirmed`.
7. Person records are scoped per case for MVP.
8. `Case.primary_root_person_id` reflects the confirmed LIRA. It may be null during eligibility research.
9. A Document always has four non-nullable phase status fields. All four are initialized at creation with their phase default: official_copy → Not Started, amendment → Unknown, apostille → Not Started, translation → Not Started.
10. Document phase status transitions are free — any status to any status within the same phase. No state machine is enforced.
11. A phase status of "Not Required" (progress_bucket: complete) signifies that phase does not apply to this document. For the official_copy phase, "Not Required" means the entire document is not needed for the case.
12. A `phase` of `any` on a DocumentStatus means the status appears in all four phase pickers.
13. Document progress computation is deferred post-MVP. When implemented, it will derive an overall progress bucket from the four phase statuses using `DocumentStatus.progress_bucket`.
14. System DocumentStatus entries cannot be edited or deleted by users.
15. User-defined DocumentStatus entries are global (shared across all cases). The admin UI for creating them is deferred post-MVP.
16. Files are uploaded directly to object storage via the backend using multipart form upload. Presigned URLs are not used for MVP.
17. Object storage buckets must have server-side encryption (SSE) enabled. This is a deployment requirement, not enforced in application code.
18. ActivityLog entries are append-only. They are never updated or deleted.
19. `ActivityLog.entity_name` is captured at the time of the action and is not updated if the entity is later renamed or deleted.
20. `ActivityLog.changes` stores field-level diffs for `updated` actions only; null for `created` and `deleted`.
21. Activity log entries are written for all write operations on case-scoped entities (cases, people, person relationships, life events, documents, file attachments, claim lines).

### Soft-delete and Trash

22. Case, Person, LifeEvent, Document, and ClaimLine have a `deleted_at` column. A null value means active; a non-null value means trashed. PersonRelationship and FileAttachment have no soft-delete.
23. All read queries filter `WHERE deleted_at IS NULL`. Trashed entities are excluded from normal results and only appear in the trash view.
24. Only directly-deleted entities appear in the trash. Children of a trashed entity are hidden implicitly — their own `deleted_at` remains null and they do not appear in the trash independently.
25. A trashed entity and all its implicitly-hidden children are frozen: no edits, no reassignment, no new children may be added until the parent is restored.
26. Restoring a trashed entity automatically restores all implicitly-hidden children (they were never independently deleted).
27. Trashed entities are permanently hard-deleted after 30 days. All descendant entities are permanently deleted via DB-level `ON DELETE CASCADE` foreign keys.
28. If a LifeEvent is permanently deleted, its Documents are also permanently deleted (CASCADE). If the user wants a Document to survive the deletion of its LifeEvent, they must reassign the Document to another event (or ungroup it) before deleting the event.
29. If a user explicitly deletes a child entity while its parent is still active (e.g., deletes a Document while the Person and LifeEvent are active), that child appears in the trash independently and follows its own 30-day clock.
30. PersonRelationship uses hard-delete only. Deleted relationships are gone immediately with no trash period.

### Reassignment

31. A LifeEvent may be reassigned to a different Person within the same Case. Cross-case reassignment is not permitted.
32. A Document may be reassigned to a different LifeEvent within the same Case, including one belonging to a different Person. When the target event belongs to a different Person, `person_id` and `life_event_id` are updated atomically in a single transaction.
33. A Document's `life_event_id` may be set to null to ungroup it from any event, leaving it associated with its Person only.
34. Reassignment is not permitted if the entity being moved or its target parent is trashed.
35. Reassignment is logged as an `updated` action in ActivityLog with the old and new parent ID(s) captured in `changes`.
36. PersonRelationship graph validation is enforced in the application layer: a Person may have at most 2 parents, and no Person may be both a direct ancestor and a direct descendant of the same Person (no circular relationships).

---

## MoSCoW

### Must (MVP)
- User, Case, ClaimLine, Person, PersonRelationship, LifeEvent, Document, FileAttachment, DocumentStatus, ActivityLog
- Multi-user authentication (shared access, all users see all data, no per-user permissions); session-based
- Activity log data capture on all write operations (display deferred)
- File upload to S3-compatible storage via multipart form, server-side encryption at rest
- Case overview (ClaimLine status summary + LifeEvents without documents)
- Document progress computation deferred post-MVP (see Could)
- ZIP export of canonical files per case
- ZIP export of all files per case

### Could (post-MVP)
- Activity feed UI (per-case feed; per-entity feed)
- Checklist/Task entity for lightweight user-driven progress tracking
- Document progress computation (derive overall progress bucket from four phase statuses)
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
