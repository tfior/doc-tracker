# API Conventions

## Base URL

All endpoints are prefixed with `/api/v1`.

## Resource URLs

Resources are nested under their owning case. Maximum nesting depth is three levels.

```
/api/v1/cases
/api/v1/cases/:caseId
/api/v1/cases/:caseId/people
/api/v1/cases/:caseId/people/:personId
/api/v1/cases/:caseId/claim-lines
/api/v1/cases/:caseId/claim-lines/:claimLineId
/api/v1/cases/:caseId/life-events
/api/v1/cases/:caseId/life-events/:lifeEventId
/api/v1/cases/:caseId/documents
/api/v1/cases/:caseId/documents/:documentId
/api/v1/cases/:caseId/documents/:documentId/attachments
/api/v1/cases/:caseId/documents/:documentId/attachments/:attachmentId
/api/v1/cases/:caseId/exports
/api/v1/auth/session
```

**Conventions:**
- All IDs are UUIDs
- Everything is case-scoped — authorization is checked at the case level and all child resources are implicitly covered
- `PersonRelationship` is managed through the people endpoints, not a separate route
- Exports are a sub-resource of cases, not top-level

## HTTP Methods

| Method | Usage |
|---|---|
| `GET` | Fetch a resource or list |
| `POST` | Create a resource |
| `PATCH` | Partial update (only send changed fields) |
| `DELETE` | Remove a resource |
| `PUT` | Not used |

## Response Shape

### Single object — bare
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Rossi Family Case",
  "status": "active",
  "created_at": "2026-04-11T10:00:00Z",
  "updated_at": "2026-04-11T10:00:00Z"
}
```

### List — minimal envelope
```json
{
  "items": [ ... ],
  "total": 42,
  "page": 1,
  "per_page": 50
}
```

`total` is always present. Default `per_page` is 50. Clients may pass `?page=1&per_page=50` as query parameters.

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

**Common error codes:**

| Code | HTTP Status | Meaning |
|---|---|---|
| `not_found` | 404 | Resource does not exist or is not accessible |
| `invalid_input` | 400 | Request body failed validation |
| `unauthorized` | 401 | No valid session |
| `forbidden` | 403 | Session valid but action not permitted |
| `conflict` | 409 | State conflict (e.g. duplicate) |
| `internal` | 500 | Unexpected server error |

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
