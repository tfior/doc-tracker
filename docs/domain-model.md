# Domain Model

## Field Types

Two date types are used in the entity tables below, in addition to the usual scalar types.

| Type | Meaning |
|---|---|
| partial date | A date known to year, year-month, or full-date precision: `1890`, `1890-03`, or `1890-03-15`. |
| qualified date | A partial date plus a qualifier (`exact`, `about`, `before`, `after`, `between`) and, for `between` only, a second partial date marking the end of the range. Stored as three columns: `<field>`, `<field>_qualifier`, `<field>_end`. |

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
An append-only record of a write action performed by a user. Captured on every create, update, delete, restore, and permanent delete across all case-scoped entities, in the same database transaction as the change it records. Display (activity feed UI) is deferred post-MVP; data capture is required from Milestone 2 onward.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| user_id | uuid | FK → User |
| action | enum | created, updated, deleted (moved to trash), restored, permanently_deleted |
| entity_type | enum | case, person, person_relationship, life_event, document, file_attachment, claim_line |
| entity_id | uuid | ID of the affected entity |
| entity_name | text | Human-readable label at the time of the action (e.g. "Birth life event for Giuseppe Rossi", "Italian Birth Certificate") |
| changes | jsonb? | Field-level diffs for updates, listing only fields whose value changed: `[{"field": "first_name", "from": "Mario", "to": "Luigi"}]`; null for all other actions |
| created_at | timestamp | |

---

### CaseDeletionRecord
A permanent record that a Case was permanently deleted. Written in the same transaction as the deletion, whether a user deleted the case or the 30-day purge did. It has no foreign key to Case, so it survives the deletion, and it is never updated or deleted. It identifies the case and nothing more: no personal data beyond the case title is stored.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | ID of the deleted Case; not a foreign key |
| case_title | text | title at the time of deletion |
| deleted_by | uuid? | FK → User; null when deleted by the automatic purge |
| deleted_at | timestamp | |
| people_count | integer | People deleted with the case |
| life_events_count | integer | LifeEvents deleted with the case |
| documents_count | integer | Documents deleted with the case |
| files_count | integer | FileAttachments deleted with the case |

---

### Case
The top-level container for a document collection effort. One case may have multiple applicants and multiple candidate lineage lines.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| title | text | |
| status | enum | active, archived, complete |
| primary_root_person_id | uuid? | FK → Person; nullable until the root ancestor is confirmed |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### ClaimLine
One candidate lineage path through the case's person graph. Always present, even for simple cases with a single root ancestor (status: eligible). Supports mid-case pivots and multi-line eligibility research without structural rebuilds.

`root_person_id` is the root ancestor: the lineage-relevant ancestor at the top of the line. It is labelled "Root Ancestor" in the UI. The full ordered path from root ancestor to applicant is a post-MVP addition (see `ClaimLineMember` in Could).

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
A person in the lineage graph, scoped per case for MVP. `first_name` and `last_name` are working identity — not verified facts. Recorded name data lives on Document. Persons cannot be reassigned across Cases.

A Person has no birth or death fields of its own. Birth date, birth place, and death date are the `event_date` and `event_place` of the Person's birth and death LifeEvents. The person form shows and edits them, creating or updating those LifeEvents behind the scenes.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| first_name | text | |
| last_name | text | |
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
A significant event in a person's life that may require one or more documents to prove. The in-line subject (`person_id`) is the lineage-relevant person. A divorce is recorded on the marriage event it ended, not as a separate event.

A spouse is usually not a Person record: their details are stored as flat fields on the marriage event. When the spouse is themselves in a line being explored, they are a Person, and the marriage event links to them through `spouse_person_id`. There is then one marriage event and one set of Documents for the marriage, shown under both people. On those Documents the second spouse is recorded as a DocumentNamedPerson with role `spouse`, linked to their Person.

LifeEvent dates are working beliefs, not recorded facts, so they are qualified dates that can start approximate and be narrowed as documents arrive.

A LifeEvent may be reassigned to a different Person within the same Case. Cross-case reassignment is not permitted.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| person_id | uuid | FK → Person (in-line subject); reassignable within same Case |
| event_type | enum | birth, marriage, death, naturalization, immigration, other |
| event_date | qualified date? | |
| event_place | text? | |
| spouse_person_id | uuid? | FK → Person in the same Case; marriage events only; set when the spouse is a Person in the case |
| spouse_name | text? | marriage events only; used when `spouse_person_id` is null |
| spouse_birth_date | qualified date? | marriage events only; used when `spouse_person_id` is null |
| spouse_birth_place | text? | marriage events only; used when `spouse_person_id` is null |
| divorce_date | qualified date? | marriage events only; set when the marriage ended in divorce |
| divorce_place | text? | marriage events only; where the divorce was granted |
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

The `apostille` phase is labelled "Apostille / Legalization" in the UI. It also covers consular legalization, the equivalent step for countries outside the Apostille Convention.

**Deferred:** Admin UI for managing user-defined statuses. Custom statuses will specify a phase and a progress_bucket. The table is ready for it; the UI is post-MVP.

---

### Document
A specific official record belonging to a Person, optionally grouped under a LifeEvent. `recorded_*` fields capture data exactly as recorded in the document, as shown on the current submission file, in the original document's original language only. Translations and apostilles do not contribute to recorded metadata. In the UI these fields sit under the heading "As recorded in document".

`details_checked` is a user's attestation that they compared the recorded details against the scan: the `recorded_*` fields and the Document's DocumentNamedPerson entries. It is labelled "Details checked against scan" in the UI. It is set by a user, independently of phase statuses, and the system records who set it, when, and which submission file was current at the time.

Every Document has a derived readiness status (see Readiness under Domain Rules). It is calculated, not stored.

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
| document_type | enum | birth_certificate, marriage_certificate, naturalization, death_certificate, non_naturalization_certificate, divorce_record, name_change_record, other |
| title | text | |
| issuing_authority | text? | |
| issue_date | partial date? | when this certified copy was issued |
| registration_date | partial date? | when the event was entered in the register (may differ from the event date) |
| recorded_given_name | text? | as recorded in the document, original language |
| recorded_surname | text? | as recorded in the document, original language |
| recorded_birth_date | partial date? | the Person's birth date as recorded in the document |
| recorded_age | integer? | the Person's age as recorded in the document, for documents that give an age ("age at last birthday") instead of, or as well as, a birth date |
| recorded_birth_place | text? | as recorded in the document |
| details_checked | boolean | default false; set by a user |
| details_checked_at | timestamp? | |
| details_checked_by | uuid? | FK → User |
| details_checked_file_id | uuid? | FK → FileAttachment; the submission file that was current when the check was made; null if none had been uploaded |
| notes | text? | AKA variants, discrepancies, amendment history narrative |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### DocumentNamedPerson
A person named in a Document other than its own subject, with what the document records about them: the parents on a birth certificate, the spouse and the couple's parents on a marriage certificate. The Document's own subject is described by the `recorded_*` fields on Document itself.

An entry may be linked to a Person in the same Case when the named person is one. The link is what allows everything recorded about a Person, in their own documents and in documents that merely mention them, to be compared. It is optional because many named people (spouses, parents outside the line) are not Person records.

Entries belong to their Document. They are edited in the document form under "Other people named in this document", saved with the Document, and deleted with it. They have no soft-delete.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| document_id | uuid | FK → Document |
| role | enum | father, mother, spouse, other; relative to the Document's subject |
| role_label | text? | required when role is `other` (e.g. "bride's father") |
| person_id | uuid? | FK → Person in the same Case; null when the named person is not in the case; set to null if that Person is permanently deleted |
| recorded_given_name | text? | as recorded in the document, original language |
| recorded_surname | text? | as recorded in the document, original language |
| recorded_birth_date | partial date? | as recorded in the document |
| recorded_age | integer? | as recorded in the document |
| recorded_birth_place | text? | as recorded in the document |
| created_at | timestamp | |
| updated_at | timestamp | |

---

### FileAttachment
A physical file attached to a Document. There are two kinds.

A **submission** file is a full scan of the document as it would be submitted with the application. The physical document grows as it is apostilled and translated, and it is re-scanned in full at each stage, so each upload is a new version of the same thing. The newest upload is the current version; earlier versions are retained and any of them can be restored, which makes it current again without deleting anything. A Document has at most one current submission file.

A **supporting** file is anything else worth keeping with the document, such as evidence or request forms. A Document may have any number of them. They have no versions.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| document_id | uuid | FK → Document |
| storage_key | text | object storage path |
| filename | text | |
| content_type | text | |
| size_bytes | integer | |
| kind | enum | submission, supporting |
| is_current | boolean | submission files only; at most one true per Document, enforced by the database |
| notes | text? | |
| uploaded_by | uuid | FK → User |
| uploaded_at | timestamp | |

---

## Domain Rules

### General

1. `recorded_*` fields always reflect the original document in its original language. Never populate from translations or apostilles.
2. `Document.details_checked` is set by a user. The only system action that clears it is the upload of a new submission file that the uploader flags as changing the recorded details.
3. A divorce is recorded on the marriage LifeEvent it ended (`divorce_date`, `divorce_place`); there is no divorce event type.
4. PersonRelationship is parent-child only.
5. ClaimLine is always present. A case with a single root ancestor has one ClaimLine.
6. Person records are scoped per case for MVP.
7. `Case.primary_root_person_id` reflects the confirmed root ancestor. It may be null during eligibility research.
8. A Person's birth and death facts live on LifeEvents. A Person has at most one birth event and at most one death event.

### Marriages and Spouses

9. A spouse is a Person record only when they are in a line being explored. Otherwise their details live on the marriage LifeEvent as flat fields (`spouse_name`, `spouse_birth_date`, `spouse_birth_place`).
10. When both spouses are Persons in the Case, the marriage is a single LifeEvent with `person_id` set to one and `spouse_person_id` set to the other. The flat spouse fields are not used; the spouse's name and birth details come from the linked Person.
11. `spouse_person_id` must refer to a Person in the same Case, other than the event's `person_id`.
12. A Person's LifeEvents are those where they are the `person_id` or the `spouse_person_id`. A shared marriage and its Documents appear under both people.
13. A flat-field spouse can be promoted to a Person: a Person is created from the flat fields and the marriage event is linked to them, in a single transaction.
14. A shared marriage is hidden, frozen, or permanently deleted along with a Person only when that is true of both spouses. If one spouse is trashed, the marriage stays active under the other. If one spouse is permanently deleted, the marriage and its Documents stay with the surviving spouse, and the deleted spouse's name is kept in `spouse_name`.

### Document Phases

15. A Document always has four non-nullable phase status fields. All four are initialized at creation with their phase default: official_copy → Not Started, amendment → Unknown, apostille → Not Started, translation → Not Started.
16. Document phase status transitions are free — any status to any status within the same phase. No state machine is enforced.
17. A phase status of "Not Required" (progress_bucket: complete) signifies that phase does not apply to this document. For the official_copy phase, "Not Required" means the entire document is not needed for the case.
18. A `phase` of `any` on a DocumentStatus means the status appears in all four phase pickers.
19. System DocumentStatus entries cannot be edited or deleted by users.
20. User-defined DocumentStatus entries are global (shared across all cases). The admin UI for creating them is deferred post-MVP.

### Readiness

21. Every Document has a readiness status derived from its phase statuses and `details_checked`:

    | Status | Rule |
    |---|---|
    | not_needed | The official_copy phase is "Not Required" |
    | ready | The document is needed, all four phases are in the `complete` progress bucket (Complete or Not Required), and `details_checked` is true |
    | not_started | The document is needed and all four phases are in the `not_started` progress bucket |
    | in_progress | Anything else |

22. A needed Document that is not ready reports why: each phase that is not complete, and whether the details have been checked.
23. Readiness does not depend on whether a file has been uploaded. A ready Document with no submission file is reported by the Application Files export.
24. Case progress is the count of the case's needed Documents in each readiness status.
25. The outstanding list for a case contains every needed Document that is not ready. Each appears once, under the earliest thing it is waiting on, in this order: official copy, amendment, apostille, translation, details check. The list can be grouped by that stage or by Person.

### Files

26. Files are uploaded directly to object storage via the backend using multipart form upload. Presigned URLs are not used for MVP.
27. Object storage buckets must have server-side encryption (SSE) enabled. This is a deployment requirement, not enforced in application code.
28. Uploading a submission file makes it the current version. Earlier versions are retained, never deleted, and any of them can be restored to current.
29. A Document has at most one current submission file. This is enforced by the database.
30. When uploading a submission file, the user may flag it as changing the recorded details. Doing so clears `details_checked`.
31. Uploading a file never changes a phase status.

### Exports

32. The Application Files export for a case contains the current submission file of every ready Document, and nothing else.
33. The Application Files export is accompanied by a single summary covering every needed Document in the case: each Document it included, with the file exported for it, and each Document it left out, with the reason (not ready, or ready but with no submission file).
34. The All Files export for a case contains every FileAttachment of every Document: all submission file versions and all supporting files.

### Dates

35. A partial date may be a year, a year and month, or a full date. A less precise date is compatible with any more precise date that falls within it: `1890` is compatible with `1890-03-15`.
36. Working dates are qualified dates. These are `LifeEvent.event_date`, `LifeEvent.spouse_birth_date`, and `LifeEvent.divorce_date`.
37. Document dates (`issue_date`, `registration_date`, `recorded_birth_date`) are partial dates with no qualifier. They transcribe what the document says.
38. A recorded age is stored as the number the document gives, never converted to a birth year. An age implies a twelve-month birth window that usually spans two calendar years. When the Document's LifeEvent has an exact full date, the UI shows that window next to the age: age 25 at an event on 1921-06-10 implies a birth between 1895-06-11 and 1896-06-10.

### Recorded Details

39. A Document's recorded details are its own `recorded_*` fields, which describe its subject, and its DocumentNamedPerson entries, which describe the other people it names. `details_checked` attests to all of them.
40. A DocumentNamedPerson may be linked only to a Person in the same Case, and not to the Document's own subject.
41. Everything recorded about a Person is the `recorded_*` fields of that Person's own Documents together with every DocumentNamedPerson entry linked to them. The person screen shows these side by side, one row per Document, with the recorded name, the birth date or the age with its implied birth window, and the birth place. Comparison is by eye; automatic discrepancy flagging is deferred post-MVP.
42. Changes to a Document's DocumentNamedPerson entries are logged as part of that Document's `updated` ActivityLog entry.

### Activity Log

43. ActivityLog entries are append-only. They are never updated. They are removed only when their Case is permanently deleted.
44. `ActivityLog.entity_name` is captured at the time of the action and is not updated if the entity is later renamed or deleted.
45. `ActivityLog.changes` stores field-level diffs for `updated` actions only, with the old and new value of each field whose value changed. It is null for all other actions.
46. Activity log entries are written for all write operations on case-scoped entities (cases, people, person relationships, life events, documents, file attachments, claim lines), including restore and permanent delete.
47. An activity log entry is written in the same database transaction as the change it records. If either fails, neither is saved.

### Soft-delete and Trash

48. Case, Person, LifeEvent, Document, and ClaimLine have a `deleted_at` column. A null value means active; a non-null value means trashed. PersonRelationship and FileAttachment have no soft-delete.
49. All read queries filter `WHERE deleted_at IS NULL`. Trashed entities are excluded from normal results and only appear in the trash view.
50. Only directly-deleted entities appear in the trash. Children of a trashed entity are hidden implicitly — their own `deleted_at` remains null and they do not appear in the trash independently.
51. A trashed entity and all its implicitly-hidden children are frozen: no edits, no reassignment, no new children may be added until the parent is restored.
52. Restoring a trashed entity automatically restores all implicitly-hidden children (they were never independently deleted).
53. Trashed entities are permanently hard-deleted after 30 days. All descendant entities are permanently deleted via DB-level `ON DELETE CASCADE` foreign keys.
54. Permanently deleting a Document, directly or through a parent, also deletes its files from object storage.
55. Permanently deleting a Case, by a user or by the 30-day purge, writes a CaseDeletionRecord in the same transaction. The case's ActivityLog entries are deleted with it.
56. If a LifeEvent is permanently deleted, its Documents are also permanently deleted (CASCADE). If the user wants a Document to survive the deletion of its LifeEvent, they must reassign the Document to another event (or ungroup it) before deleting the event.
57. If a user explicitly deletes a child entity while its parent is still active (e.g., deletes a Document while the Person and LifeEvent are active), that child appears in the trash independently and follows its own 30-day clock.
58. PersonRelationship uses hard-delete only. Deleted relationships are gone immediately with no trash period.

### Reassignment

59. A LifeEvent may be reassigned to a different Person within the same Case. Cross-case reassignment is not permitted.
60. A Document may be reassigned to a different LifeEvent within the same Case, including one belonging to a different Person. When the target event belongs to a different Person, `person_id` and `life_event_id` are updated atomically in a single transaction.
61. A Document's `life_event_id` may be set to null to ungroup it from any event, leaving it associated with its Person only.
62. Reassignment is not permitted if the entity being moved or its target parent is trashed.
63. Reassignment is logged as an `updated` action in ActivityLog with the old and new parent ID(s) captured in `changes`.
64. PersonRelationship graph validation is enforced in the application layer: a Person may have at most 2 parents, and no Person may be both a direct ancestor and a direct descendant of the same Person (no circular relationships).

---

## MoSCoW

### Must (MVP)
- User, Case, ClaimLine, Person, PersonRelationship, LifeEvent, Document, DocumentNamedPerson, FileAttachment, DocumentStatus, ActivityLog, CaseDeletionRecord
- Multi-user authentication (shared access, all users see all data, no per-user permissions); session-based
- Activity log data capture on all write operations, in the same transaction as the change (display deferred)
- File upload to S3-compatible storage via multipart form, server-side encryption at rest: a versioned submission file per document, plus supporting files
- Partial and qualified dates, and recorded ages
- Other people named in a document, and a per-person comparison of recorded details
- Marriages between two people in a case (one shared marriage event; promote a spouse to a person)
- Case overview (ClaimLine status summary + LifeEvents without documents)
- Document readiness and case progress (summary counts and outstanding list)
- Application Files export per case (ZIP of ready documents' current submission files, with a summary of every needed document: included or left out, and why)
- All Files export per case (ZIP)
- Production deployment readiness (see Milestone 4 in `milestones.md`)

### Could (post-MVP)
- Activity feed UI (per-case feed; per-entity feed)
- Checklist/Task entity for lightweight user-driven progress tracking
- Admin UI for managing user-defined DocumentStatus entries
- `ClaimLineMember` — explicit ordered path from root ancestor to applicant. A `claim_line_members` table with `(claim_line_id, person_id, position)` making the specific genealogical route unambiguous. Required when multiple paths exist between root ancestor and applicant due to branching parentage. Current MVP stores only `root_person_id`; the full path is implied by the `person_relationships` graph.
- Async export job records with file manifest
- DocumentNameRecord — structured AKA / name variants per document (several recorded names for the same person)
- NameDiscrepancy records with severity and resolution status
- DocumentAmendment structured tracking
- `canonical_person_id` on Person for cross-case linking
- Global person/document scope across cases
- Second case type for eligibility workflow
- Automatic discrepancy flagging across documents
- Presigned URL upload flow (if direct upload becomes a bottleneck)
- User registration flow
- User-management UI (add, disable, reset password)
- Roles with per-case access: admins see all cases, case managers see only their own
- Read-only client access to their own case, to track progress
- Two-factor login
- Case list search and filtering
- User-managed document types

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
