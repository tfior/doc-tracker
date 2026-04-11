# Domain Model

## Entities

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

---

### ClaimLine
One candidate lineage path through the case's person graph. Always present, even for simple single-LIRA cases (status: confirmed). Supports mid-case pivots and multi-line eligibility research without structural rebuilds.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| root_person_id | uuid | FK → Person |
| status | enum | active, suspended, eliminated, confirmed |
| notes | text? | |
| created_at | timestamp | |
| updated_at | timestamp | |

---

### Person
A person in the lineage graph, scoped per case for MVP. `first_name` and `last_name` are working identity — not verified facts. Verified name data lives on Document.

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

**Deferred:** `canonical_person_id` (nullable FK → Person) for cross-case linking. Add when needed.

---

### PersonRelationship
Parent-child relationships only. Supports multiple parents per person to handle branching lineage lines. Marriages and other relationships are captured via LifeEvent, not here.

| Field | Type | Notes |
|---|---|---|
| person_id | uuid | FK → Person |
| parent_id | uuid | FK → Person |
| case_id | uuid | FK → Case |

---

### LifeEvent
A significant event in a person's life that may require one or more documents to prove. The in-line subject (`person_id`) is the lineage-relevant person. Spouse data is stored as flat fields — spouses are not Person records in the graph.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| person_id | uuid | FK → Person (in-line subject) |
| event_type | enum | birth, marriage, death, naturalization, immigration, other |
| event_date | date? | |
| event_place | text? | |
| spouse_name | text? | marriage events only |
| spouse_birth_date | date? | marriage events only |
| spouse_birth_place | text? | marriage events only |
| notes | text? | |
| created_at | timestamp | |
| updated_at | timestamp | |

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

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| person_id | uuid | FK → Person |
| life_event_id | uuid? | FK → LifeEvent |
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

---

## MoSCoW

### Must (MVP)
- Case, ClaimLine, Person, PersonRelationship, LifeEvent, Document, FileAttachment, DocumentStatus
- Authentication (single-owner, session-based)
- File upload to S3-compatible storage via multipart form, server-side encryption at rest
- Case overview and progress tracking (ClaimLine status + document progress buckets + LifeEvents without documents)
- ZIP export of canonical files per case
- ZIP export of all files per case

### Could (post-MVP)
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

### Won't (MVP)
- RequirementSpec / CaseRequirement — adds complexity for little MVP benefit; checklist covers the need
- Multi-user collaboration
- Public sharing portals
- Google Drive / GEDCOM / GRAMPS integration
- Full family tree builder
- Conditional requirement logic / rules engine
- OCR / AI extraction
- Workflow automation
- Microservices
