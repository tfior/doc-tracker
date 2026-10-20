# API Conventions

## Base URL

All endpoints are prefixed with `/api/v1`.

## Resource URLs

Resources are nested under their owning case.

```
/api/v1/auth/session

/api/v1/cases
/api/v1/cases/:caseId
/api/v1/cases/:caseId/people
/api/v1/cases/:caseId/people/:personId
/api/v1/cases/:caseId/people/:personId/recorded-details
/api/v1/cases/:caseId/claim-lines
/api/v1/cases/:caseId/claim-lines/:claimLineId
/api/v1/cases/:caseId/applications
/api/v1/cases/:caseId/applications/:applicationId
/api/v1/cases/:caseId/applications/:applicationId/export
/api/v1/cases/:caseId/life-events
/api/v1/cases/:caseId/life-events/:lifeEventId
/api/v1/cases/:caseId/documents
/api/v1/cases/:caseId/documents/:documentId
/api/v1/cases/:caseId/documents/:documentId/files
/api/v1/cases/:caseId/documents/:documentId/files/:fileId
/api/v1/cases/:caseId/exports/all-files
/api/v1/cases/:caseId/trash

/api/v1/trash
/api/v1/document-statuses
```

**Conventions:**
- All IDs are UUIDs
- Everything belonging to a case is nested under it. There is no per-case access control for MVP, but when it is added, one check at the case level will cover every child resource
- Files and exports are case-scoped for the same reason. No file is addressable by its ID alone
- `PersonRelationship` is managed through the people endpoints, not a separate route
- `DocumentNamedPerson` entries are part of their Document, read and written with it, not a separate route
- A ClaimLine's members are part of it, returned with it, not a separate route. A ClaimLine is created with an applicant and a root ancestor. When more than one path exists between them, the request must also give `path`, the person IDs in order; without it, the request is refused with the available paths listed
- An Application's applicants are part of the Application, read and written with it, not a separate route
- The Application Files export is a sub-resource of its Application. The All Files export is a sub-resource of the case. Neither is top-level
- `GET /api/v1/cases` accepts `?status=` (`active`, `on_hold`, `complete`), `?archived=true` to list archived cases instead of current ones, and `?q=` to search by title
- `DocumentStatus` entries are workspace-global and not nested under a case
- `GET /api/v1/document-statuses` accepts an optional `?phase=` query parameter to filter by phase (`official_copy`, `amendment`, `apostille`, `translation`, `any`). Omitting it returns all statuses.
- `GET /api/v1/cases/:caseId/trash` lists the trashed entities within a case. `GET /api/v1/trash` lists trashed cases.
- `GET /api/v1/cases/:caseId/people/:personId/recorded-details` returns everything recorded about a Person, one row per Document

### Actions

An operation that is not a plain create, read, update, or delete is a sub-path of the resource it acts on.

| Action | Request |
|---|---|
| Restore from trash | `POST /api/v1/cases/:caseId/<resource>/:id/restore` (and `POST /api/v1/cases/:caseId/restore`) |
| Permanently delete | `DELETE /api/v1/cases/:caseId/<resource>/:id/permanent` (and `DELETE /api/v1/cases/:caseId/permanent`) |
| Archive or unarchive a case | `POST /api/v1/cases/:caseId/archive`, `POST /api/v1/cases/:caseId/unarchive` |
| Promote a flat-field spouse to a Person | `POST /api/v1/cases/:caseId/life-events/:lifeEventId/promote-spouse` |
| Make an earlier submission file current | `POST /api/v1/cases/:caseId/documents/:documentId/files/:fileId/restore` |

## HTTP Methods

| Method | Usage |
|---|---|
| `GET` | Fetch a resource or list |
| `POST` | Create a resource, or perform an action |
| `PATCH` | Partial update (only send changed fields) |
| `DELETE` | Move a resource to the trash, or permanently delete it via `/permanent` |
| `PUT` | Not used |

## Updates

- **One way to update each entity.** `PATCH` on a resource accepts every field a user can change on it, including its parent and, for a Document, its four phase statuses and its DocumentNamedPerson entries. There are no single-purpose endpoints for changing one aspect of a resource.
- **Partial.** Only fields present in the request body are changed. A field sent as `null` is cleared. A field that is absent is left alone.
- **All or nothing.** A request is applied in one database transaction. If any part of it is invalid or fails, nothing is saved.
- **One request per save.** A form's Save sends a single request. The frontend does not split a save across several requests.

## Concurrent Edits

Case, Person, LifeEvent, Document, and ClaimLine each carry an integer `version` that increases on every change.

- Every response includes the resource's current `version`.
- A `PATCH` or `DELETE` must include the `version` the client last read.
- If the resource has changed since, the request is refused with `409` and the code `conflict`, and nothing is saved. The client reloads the resource and the user reapplies their change.

This prevents one user's save from silently overwriting another's.

## Response Shape

### Single object — bare
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Rossi Family Case",
  "status": "active",
  "version": 3,
  "created_at": "2026-04-11T10:00:00Z",
  "updated_at": "2026-04-11T10:00:00Z"
}
```

### List — minimal envelope
```json
{
  "items": [ ... ],
  "total": 42
}
```

`total` is always present.

### Pagination

- **Lists inside a case are not paginated.** A case's people, life events, documents, and claim lines are returned in full, in one response. They are bounded by the size of a case, and every screen needs all of them.
- **The case list is paginated.** `GET /api/v1/cases` accepts `?page=1&per_page=50`. Default `per_page` is 50; the maximum is 200. Its envelope adds `page` and `per_page`.

### Derived values

Responses include every derived value the interface needs. Clients do not recompute them.

- Every Document carries `readiness`: its status, the stage it is waiting on, and the reasons it is not ready. Reasons are machine-readable codes; the frontend supplies the wording.

  ```json
  "readiness": {
    "status": "in_progress",
    "waiting_on": "apostille",
    "reasons": ["apostille_incomplete", "translation_incomplete", "details_not_checked"]
  }
  ```

- Every Document also carries, for each phase, the date it entered its current status; the age of its current copy, when the issue date allows; and the number of copies required.
- Every Application carries its Documents, in packet order, and its status with the number ready out of the total.
- Every Case carries `progress`, in both the case list and the case detail: the count of its Documents in each readiness status.

  ```json
  "progress": { "ready": 5, "in_progress": 6, "not_started": 1, "not_needed": 2 }
  ```

- A recorded age is returned with the birth window it implies, when that can be calculated.

## Error Shape

All errors return a consistent structure regardless of status code:

```json
{
  "error": {
    "code": "not_found",
    "message": "Case not found"
  }
}
```

Validation errors also list each field at fault, with a message a user can act on:

```json
{
  "error": {
    "code": "invalid_input",
    "message": "Some fields are invalid",
    "fields": [
      { "field": "event_date", "code": "invalid_partial_date", "message": "Enter a year, a year and month, or a full date" },
      { "field": "title", "code": "required", "message": "Title is required" }
    ]
  }
}
```

- `message` is always written for the user, not the developer.
- Each entry in `fields` carries a stable `code` alongside its message, so the wording can later be translated without changing the API.
- `fields` is present whenever the problem can be tied to specific fields. A request with several invalid fields reports all of them, not only the first.
- The frontend shows field messages beside the fields and the top-level message otherwise. It does not replace a server message with a generic one.

**Common error codes:**

| Code | HTTP Status | Meaning |
|---|---|---|
| `not_found` | 404 | Resource does not exist or is not accessible |
| `invalid_input` | 400 | Request body failed validation |
| `unauthorized` | 401 | No valid session |
| `forbidden` | 403 | Session valid but action not permitted |
| `conflict` | 409 | State conflict: the resource changed since the client read it, or a duplicate |
| `internal` | 500 | Unexpected server error |

## Authentication

- `POST /api/v1/auth/session` logs in, `GET` checks the current session, `DELETE` logs out.
- The session is carried in a cookie that is HTTP-only and `SameSite=Lax`. In production it is also HTTPS-only.
- Every other endpoint under `/api/v1` requires a valid session and returns `401` with the code `unauthorized` without one.

## Files

- Uploads use multipart form data and go through the backend. Presigned URLs are not used.
- Downloads are streamed through the backend from a case-scoped URL.
- A file can be requested for viewing or for download. For viewing, it is served so that the browser displays it in a tab.
- A submission file must be a PDF; a supporting file must be a PDF or an image. The server checks the file's content, not only its name, and refuses anything else with `invalid_input`.
- The maximum upload size is a deployment setting. A file over the limit is refused with `invalid_input` and a message stating the limit.
- Exports are ZIP files. The names of the files inside are generated by the server.

## Dates

Dates that describe people, events, and documents are not timestamps. They may be known only in part, so they are sent as text, never as a full calendar date with invented parts.

**Partial date** — a string in one of three forms:

```json
"1890"
"1890-03"
"1890-03-15"
```

**Qualified date** — an object holding a partial date, a qualifier, and for `between` only, a second partial date:

```json
{ "value": "1890-03-15", "qualifier": "exact" }
{ "value": "1890", "qualifier": "about" }
{ "value": "1915", "qualifier": "before" }
{ "value": "1888-06", "qualifier": "after" }
{ "value": "1890", "qualifier": "between", "end": "1892" }
```

- `qualifier` is one of `exact`, `about`, `before`, `after`, `between`.
- `end` is required for `between` and not allowed otherwise.
- An unknown date is `null`, not an object with empty parts.
- Which fields are partial dates and which are qualified dates is defined in the domain model.

## Timestamps

- All timestamps are in UTC, formatted as RFC 3339 (`2026-04-11T10:00:00Z`)
- Request bodies may omit timezone offset — the server always interprets as UTC
- `created_at` and `updated_at` are set server-side and never accepted from the client

## Naming

- JSON field names: `snake_case`
- URL path segments: `kebab-case`
- Enum values: `snake_case` strings (e.g. `"active"`, `"not_found"`, `"birth_certificate"`)

## Versioning

The API is versioned via URL prefix (`/api/v1`). Breaking changes require a new version prefix. Non-breaking additions (new fields, new endpoints) do not require a version bump.
