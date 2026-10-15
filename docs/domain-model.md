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
| entity_type | enum | case, person, person_relationship, life_event, document, file_attachment, claim_line, application |
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
The top-level container for a document collection effort. A case explores one or more candidate root ancestors through its ClaimLines and pursues exactly one of them. It may have several applicants, grouped into Applications.

Archiving is a state, not a status. An archived case keeps its status, is hidden from the default case list, and is read-only until it is unarchived.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| title | text | |
| status | enum | active, on_hold, complete |
| pursued_claim_line_id | uuid? | FK → ClaimLine; the line this case pursues; null until one is chosen |
| notes | text? | |
| archived_at | timestamp? | null = not archived; set = archived |
| version | integer | increases on every change; used to detect concurrent edits |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### ClaimLine
One candidate root ancestor and the line descending from them. A new case has none; a simple case ends up with one. Supports mid-case pivots and multi-line eligibility research without structural rebuilds. Several ClaimLines in a case may be eligible, but the case pursues only one.

`root_person_id` is the root ancestor: the lineage-relevant ancestor at the top of the line. It is labelled "Root Ancestor" in the UI. The path from the root ancestor down to any applicant is not stored; it is derived from the parent-child relationships.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| root_person_id | uuid | FK → Person; immutable after creation |
| status | enum | not_yet_researched, researching, paused, ineligible, eligible |
| notes | text? | |
| version | integer | increases on every change; used to detect concurrent edits |
| created_at | timestamp | |
| updated_at | timestamp | |
| deleted_at | timestamp? | null = active; set = trashed |

---

### Application
One submission: one venue, one physical set of documents, and one or more applicants. A case may have several, for example relatives who must apply at different consulates, or groups limited by a venue's cap on applicants per document set. Every Application in a case relies on the case's pursued ClaimLine.

An Application's Documents and its status are derived, not stored (see Applications under Domain Rules). Applications have no soft-delete: deleting one is immediate and permanent, and the UI says so before the user confirms.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| name | text | e.g. "New York consulate: Sofia and children" |
| venue_type | enum | consulate, comune, court, other |
| venue | text? | e.g. "Consulate General, Chicago" |
| notes | text? | |
| version | integer | increases on every change; used to detect concurrent edits |
| created_at | timestamp | |
| updated_at | timestamp | |

---

### ApplicationApplicant
The applicants of an Application. A bare join table with no soft-delete.

| Field | Type | Notes |
|---|---|---|
| application_id | uuid | FK → Application |
| person_id | uuid | FK → Person in the same Case |
| case_id | uuid | FK → Case |

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
| version | integer | increases on every change; used to detect concurrent edits |
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
| version | integer | increases on every change; used to detect concurrent edits |
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

Each of the four phases carries three things: its status, the date it entered that status, and an optional reference such as an order or case number.

Several values are derived, not stored: the readiness status (see Readiness under Domain Rules), the age of the current copy, and the number of copies required.

A Document may be reassigned to a different LifeEvent within the same Case, including one belonging to a different Person. When reassigned cross-person, `person_id` and `life_event_id` are updated atomically. Setting `life_event_id` to null ungrouped the Document under the Person with no event association.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| case_id | uuid | FK → Case |
| person_id | uuid | FK → Person; updated atomically when reassigned to an event under a different Person |
| life_event_id | uuid? | FK → LifeEvent; null = ungrouped; reassignable within same Case |
| official_copy_status_id | uuid | FK → DocumentStatus (phase=official_copy or any); default: Not Started |
| official_copy_status_since | date | the date the phase entered its current status; set automatically, correctable |
| official_copy_reference | text? | order, case, or tracking number for this phase's request |
| amendment_status_id | uuid | FK → DocumentStatus (phase=amendment or any); default: Unknown |
| amendment_status_since | date | the date the phase entered its current status; set automatically, correctable |
| amendment_reference | text? | order, case, or tracking number for this phase's request |
| apostille_status_id | uuid | FK → DocumentStatus (phase=apostille or any); default: Not Started |
| apostille_status_since | date | the date the phase entered its current status; set automatically, correctable |
| apostille_reference | text? | order, case, or tracking number for this phase's request |
| translation_status_id | uuid | FK → DocumentStatus (phase=translation or any); default: Not Started |
| translation_status_since | date | the date the phase entered its current status; set automatically, correctable |
| translation_reference | text? | order, case, or tracking number for this phase's request |
| document_type | enum | birth_certificate, marriage_certificate, naturalization, death_certificate, non_naturalization_certificate, divorce_record, court_order, other |
| title | text | |
| for_submission | boolean | default true; false = tracked but not part of any application packet |
| issuing_authority | text? | |
| issue_date | partial date? | when this certified copy was issued; a full date allows the copy's age to be calculated |
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
| version | integer | increases on every change; used to detect concurrent edits |
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

A **submission** file is a full scan of the document as it would be submitted with the application. The physical document grows as it is apostilled and translated, and it is re-scanned in full at each stage, so each upload is a new version of the same thing. The newest upload is the current version; earlier versions are retained and any of them can be restored, which makes it current again without deleting anything. A Document has at most one current submission file. A submission file must be a PDF.

A **supporting** file is anything else worth keeping with the document, such as evidence or request forms. A Document may have any number of them. They have no versions. A supporting file may be a PDF or an image.

| Field | Type | Notes |
|---|---|---|
| id | uuid | |
| document_id | uuid | FK → Document |
| storage_key | text | object storage path |
| filename | text | name the file was uploaded with; never used in exports |
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
5. Person records are scoped per case for MVP.
6. A Person's birth and death facts live on LifeEvents. A Person has at most one birth event and at most one death event.
7. Case, Person, LifeEvent, Document, ClaimLine, and Application each carry a `version` that increases on every change. A change must state the version it was based on. If the record has changed since, the change is refused and nothing is saved, so one user's save never silently overwrites another's.

### Cases

8. A Case pursues at most one ClaimLine, named by `pursued_claim_line_id`. It is null until a line is chosen, and may be changed to a different line.
9. Only a ClaimLine with status `eligible` can be pursued. While a ClaimLine is pursued, its status cannot be changed away from `eligible`; the Case must stop pursuing it first. An attempt to change it is refused with a message that says this, so the user knows what to do.
10. No two applicants in a Case claim through different root ancestors. This is enforced by structure: the pursued line is chosen once on the Case, and Applications do not name a line of their own.
11. An archived Case is hidden from the default case list and is read-only: nothing in it can be added, changed, or deleted until it is unarchived. Archiving does not change the Case's status, and an archived Case is never purged.

### Applications

12. An Application's applicants are Persons in the same Case.
13. An Application's Documents are derived. For each applicant, take every Person on the path from the applicant up to the root ancestor of the Case's pursued ClaimLine, both included, following parent-child relationships; where more than one path exists, all are included. The Application's Documents are the needed, for-submission Documents of those Persons, each counted once.
14. If the Case pursues no ClaimLine, or an applicant does not descend from its root ancestor, that applicant contributes no Documents, and the Application reports why, prominently, so the user can tell whether the applicant was added in error or a parent link is missing.
15. An Application's status is derived: `ready_for_submission` when it has at least one Document and every one of its Documents is ready; otherwise `preparing`, reported with the number ready out of the total.
16. The number of copies required of a Document is the number of Applications in the Case whose Documents include it. A Document's phases describe its set of copies as a whole.

### Marriages and Spouses

17. A spouse is a Person record only when they are in a line being explored. Otherwise their details live on the marriage LifeEvent as flat fields (`spouse_name`, `spouse_birth_date`, `spouse_birth_place`).
18. When both spouses are Persons in the Case, the marriage is a single LifeEvent with `person_id` set to one and `spouse_person_id` set to the other. The flat spouse fields are not used; the spouse's name and birth details come from the linked Person.
19. `spouse_person_id` must refer to a Person in the same Case, other than the event's `person_id`.
20. A Person's LifeEvents are those where they are the `person_id` or the `spouse_person_id`. A shared marriage and its Documents appear under both people.
21. A flat-field spouse can be promoted to a Person: a Person is created from the flat fields and the marriage event is linked to them, in a single transaction.
22. A shared marriage is hidden, frozen, or permanently deleted along with a Person only when that is true of both spouses. If one spouse is trashed, the marriage stays active under the other. If one spouse is permanently deleted, the marriage and its Documents stay with the surviving spouse, and the deleted spouse's name is kept in `spouse_name`.

### Document Phases

23. A Document always has four non-nullable phase status fields. All four are initialized at creation with their phase default: official_copy → Not Started, amendment → Unknown, apostille → Not Started, translation → Not Started.
24. Document phase status transitions are free — any status to any status within the same phase. No state machine is enforced.
25. A phase status of "Not Required" (progress_bucket: complete) signifies that phase does not apply to this document. For the official_copy phase, "Not Required" means the entire document is not needed for the case.
26. A `phase` of `any` on a DocumentStatus means the status appears in all four phase pickers.
27. System DocumentStatus entries cannot be edited or deleted by users.
28. User-defined DocumentStatus entries are global (shared across all cases). The admin UI for creating them is deferred post-MVP.
29. Each phase records the date it entered its current status. The date is set to the current day whenever the status changes, and a user may correct it, for example to back-date a request entered late. How long a phase has been in its status is derived from it.
30. Each phase has an optional free-text reference for the order, case, or tracking number of its request.

### Readiness

31. Every Document has a readiness status derived from its phase statuses and `details_checked`:

    | Status | Rule |
    |---|---|
    | not_needed | The official_copy phase is "Not Required" |
    | ready | The document is needed, all four phases are in the `complete` progress bucket (Complete or Not Required), and `details_checked` is true |
    | not_started | The document is needed and all four phases are in the `not_started` progress bucket |
    | in_progress | Anything else |

32. A needed Document that is not ready reports why: each phase that is not complete, and whether the details have been checked.
33. Readiness does not depend on whether a file has been uploaded. A ready Document with no submission file is reported by the Application Files export.
34. Case progress is the count of the case's needed Documents in each readiness status.
35. The outstanding list for a case contains every needed Document that is not ready. Each appears once, under the earliest thing it is waiting on, in this order: official copy, amendment, apostille, translation, details check. The list can be grouped by that stage or by Person, and sorted by how long each Document has been waiting.
36. A Document with `for_submission` set to false is tracked like any other, with its own readiness, but belongs to no Application's Documents and is left out of the Application Files export. It appears in the All Files export.
37. The age of a Document's current copy is derived from `issue_date` when that is a full date. It is shown wherever an Application's Documents are listed. The app applies no age limit; whether a copy is too old for a venue is the user's judgement.

### Files

38. Files are uploaded directly to object storage via the backend using multipart form upload. Presigned URLs are not used for MVP.
39. Object storage buckets must have server-side encryption (SSE) enabled. This is a deployment requirement, not enforced in application code.
40. Uploading a submission file makes it the current version. Earlier versions are retained, never deleted, and any of them can be restored to current.
41. A Document has at most one current submission file. This is enforced by the database.
42. A submission file must be a PDF. A supporting file must be a PDF or an image. The file's content is checked, not only its name.
43. The maximum upload size is a deployment setting.
44. When uploading a submission file, the user may flag it as changing the recorded details. Doing so clears `details_checked`.
45. Uploading a file never changes a phase status.

### Exports

46. The Application Files export is produced for one Application. It contains the current submission file of every ready Document among the Application's Documents, and nothing else.
47. The export is a ZIP containing the files and a manifest. The manifest covers every one of the Application's Documents: each Document included, with its file name and the age of its copy, and each Document left out, with the reason (not ready, or ready but with no submission file).
48. Exported files are named by the app: packet order, generation, the subject's surname and given name, and the document type, for example `04 G2 Rossi Antonio - Birth Certificate.pdf`. Generation is counted from the root ancestor, who is generation 1, so a name does not depend on which applicant is applying. The Document's title is appended for the `court_order` and `other` types.
49. Packet order runs from the root ancestor down to the applicants, and matches the order of the manifest.
50. The ZIP, and the single folder inside it, are named for the Case and the Application.
51. The All Files export for a case contains every FileAttachment of every Document: all submission file versions and all supporting files.

### Dates

52. A partial date may be a year, a year and month, or a full date. A less precise date is compatible with any more precise date that falls within it: `1890` is compatible with `1890-03-15`.
53. Working dates are qualified dates. These are `LifeEvent.event_date`, `LifeEvent.spouse_birth_date`, and `LifeEvent.divorce_date`.
54. Document dates (`issue_date`, `registration_date`, `recorded_birth_date`) are partial dates with no qualifier. They transcribe what the document says.
55. A recorded age is stored as the number the document gives, never converted to a birth year. An age implies a twelve-month birth window that usually spans two calendar years. When the Document's LifeEvent has an exact full date, the UI shows that window next to the age: age 25 at an event on 1921-06-10 implies a birth between 1895-06-11 and 1896-06-10.

### Recorded Details

56. A Document's recorded details are its own `recorded_*` fields, which describe its subject, and its DocumentNamedPerson entries, which describe the other people it names. `details_checked` attests to all of them.
57. A DocumentNamedPerson may be linked only to a Person in the same Case, and not to the Document's own subject.
58. Everything recorded about a Person is the `recorded_*` fields of that Person's own Documents together with every DocumentNamedPerson entry linked to them. The person screen shows these side by side, one row per Document, with the recorded name, the birth date or the age with its implied birth window, and the birth place. The Person's birth certificate is listed first and marked as the reference, since their other documents must agree with it. Comparison is by eye; automatic discrepancy flagging is deferred post-MVP.
59. Changes to a Document's DocumentNamedPerson entries are logged as part of that Document's `updated` ActivityLog entry.

### Activity Log

60. ActivityLog entries are append-only. They are never updated. They are removed only when their Case is permanently deleted.
61. `ActivityLog.entity_name` is captured at the time of the action and is not updated if the entity is later renamed or deleted.
62. `ActivityLog.changes` stores field-level diffs for `updated` actions only, with the old and new value of each field whose value changed. It is null for all other actions.
63. Activity log entries are written for all write operations on case-scoped entities (cases, people, person relationships, life events, documents, file attachments, claim lines, applications), including archive, restore, and permanent delete.
64. An activity log entry is written in the same database transaction as the change it records. If either fails, neither is saved.

### Soft-delete and Trash

65. Case, Person, LifeEvent, Document, and ClaimLine have a `deleted_at` column. A null value means active; a non-null value means trashed. Application, PersonRelationship, DocumentNamedPerson, and FileAttachment have no soft-delete.
66. All read queries filter `WHERE deleted_at IS NULL`. Trashed entities are excluded from normal results and only appear in the trash view.
67. Only directly-deleted entities appear in the trash. Children of a trashed entity are hidden implicitly — their own `deleted_at` remains null and they do not appear in the trash independently.
68. A trashed entity and all its implicitly-hidden children are frozen: no edits, no reassignment, no new children may be added until the parent is restored.
69. Restoring a trashed entity automatically restores all implicitly-hidden children (they were never independently deleted).
70. Trashed entities are permanently hard-deleted after 30 days. All descendant entities are permanently deleted via DB-level `ON DELETE CASCADE` foreign keys.
71. Permanently deleting a Document, directly or through a parent, also deletes its files from object storage.
72. Permanently deleting a Case, by a user or by the 30-day purge, writes a CaseDeletionRecord in the same transaction. The case's ActivityLog entries are deleted with it.
73. If a LifeEvent is permanently deleted, its Documents are also permanently deleted (CASCADE). If the user wants a Document to survive the deletion of its LifeEvent, they must reassign the Document to another event (or ungroup it) before deleting the event.
74. Before a Person or LifeEvent is moved to the trash or permanently deleted, the user is told what goes with it: for a LifeEvent, the number of its Documents; for a Person, the number of their LifeEvents and Documents, and any ClaimLine rooted at them.
75. If a user explicitly deletes a child entity while its parent is still active (e.g., deletes a Document while the Person and LifeEvent are active), that child appears in the trash independently and follows its own 30-day clock.
76. PersonRelationship uses hard-delete only. Deleted relationships are gone immediately with no trash period.

### Reassignment

77. A LifeEvent may be reassigned to a different Person within the same Case. Cross-case reassignment is not permitted.
78. A Document may be reassigned to a different LifeEvent within the same Case, including one belonging to a different Person. When the target event belongs to a different Person, `person_id` and `life_event_id` are updated atomically in a single transaction.
79. A Document's `life_event_id` may be set to null to ungroup it from any event, leaving it associated with its Person only.
80. Reassignment is not permitted if the entity being moved or its target parent is trashed.
81. Reassignment is logged as an `updated` action in ActivityLog with the old and new parent ID(s) captured in `changes`.
82. PersonRelationship graph validation is enforced in the application layer: a Person may have at most 2 parents, and no Person may be both a direct ancestor and a direct descendant of the same Person (no circular relationships).

---

## MoSCoW

### Must (MVP)
- User, Case, ClaimLine, Application, ApplicationApplicant, Person, PersonRelationship, LifeEvent, Document, DocumentNamedPerson, DocumentStatus, ActivityLog, CaseDeletionRecord
- Multi-user authentication (shared access, all users see all data, no per-user permissions); session-based
- Activity log data capture on all write operations, in the same transaction as the change (display deferred)
- Record versions, so concurrent edits are refused rather than silently overwritten
- Partial and qualified dates, and recorded ages
- Other people named in a document, and a per-person comparison of recorded details
- Marriages between two people in a case (one shared marriage event)
- Case statuses (active, on hold, complete), archiving, and case notes
- One pursued claim line per case
- Applications: applicants grouped by venue, with derived documents, status, and copies required
- Case list with status filter and title search
- Case overview (ClaimLine status summary + LifeEvents without documents)
- Document readiness and case progress (summary counts and outstanding list by stage)
- Request tracking on each phase: date entered, time waiting, reference number
- The "for submission" flag on documents
- Production deployment readiness (see the Deployable milestone in `milestones.md`)

### Should (first releases after the MVP)
- FileAttachment: file upload to S3-compatible storage via multipart form, server-side encryption at rest; a versioned PDF submission file per document, plus supporting files; a single file can be opened in the browser or downloaded
- Application Files export per application (ZIP of ready documents' current submission files under generated names, with a manifest of every document: included or left out, and why)
- All Files export per case (ZIP)

### Could (post-MVP)
- Activity feed UI (per-case feed; per-entity feed)
- Checklist/Task entity for lightweight user-driven progress tracking
- Admin UI for managing user-defined DocumentStatus entries
- `ClaimLineMember` — explicit ordered path from root ancestor to applicant, for the rare case where more than one path exists between them and only one should be used. The MVP derives the path from the `person_relationships` graph and includes every path.
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
- Case list filtering beyond status and title
- User-managed document types
- Grouping the outstanding list by person
- An action to promote a flat-field spouse to a Person in one step
- Restoring an earlier version of a submission file
- The life event form offering to add the event's expected document in the same step
- Documents attached to an application rather than a person (powers of attorney, application forms), and an Attorney File export that includes them
- A link from amended documents to the court order that amended them
- Document age limits per application, with warnings (a planned submission date, and documents marked as needing a recent copy)
- A prompt to reset the later phases when an official copy is replaced
- Expense tracking per request (fees, shipping, invoicing)
- Physical location of a document (with domestic team, with overseas team, with attorney, with client; user-defined options later)
- Copies obtained per document, and flagging a shortfall against copies required
- Outstanding list and progress per application
- Adding a document from outside the path to an application
- A strict review option requiring details to be checked by someone other than the person who entered them
- A scan viewer embedded beside the document form
- Interface languages other than English

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
