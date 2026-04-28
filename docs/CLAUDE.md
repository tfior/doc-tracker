# Claude Instructions for doc-tracker

This file contains instructions for AI-assisted development on this project. Read it at the start of every session.

## Docs Folder

The `docs/` folder is the authoritative reference for project decisions. Keep these files current — they are not historical logs, they are living documents. Update them whenever a decision is made, revised, or reversed.

### Files and What They Own

| File | Owns |
|---|---|
| `domain-model.md` | All entities, fields, relationships, enums, and domain rules. Also tracks what is explicitly deferred or out of scope. |
| `architecture.md` | Stack choices, module boundaries, repo structure, backend package structure, and architectural constraints. |
| `api-conventions.md` | HTTP API design conventions, URL patterns, request/response shapes, error formats, auth conventions. |
| `milestones.md` | Implementation milestones, acceptance criteria, and task checklists. |
| `CLAUDE.md` | This file. Instructions for AI-assisted development. |

### When to Update

- **domain-model.md** — whenever an entity is added, removed, or modified; whenever a field is added, renamed, or removed; whenever a domain rule is established or changed; whenever something is moved between MoSCoW tiers.
- **architecture.md** — whenever a stack decision is made or changed; whenever module boundaries are drawn or revised; whenever a new package or directory is added to the canonical structure.
- **api-conventions.md** — whenever an API endpoint is designed or changed; whenever a new convention is established.
- **milestones.md** — whenever a milestone is defined, revised, or completed; whenever a task within a milestone is checked off.

IMPORTANT: Only update these files (including this CLAUDE.md file) after permission is explicitly granted to do so. Do not wait until the end of a session to ask for permission. The documents should be kept up-to-date in realtime. But again, ask before making changes to these documents in EVERY CASE, even when you have permission to edit other files without asking first.

### How to Update

- Replace outdated content directly. These are not changelogs — do not append "as of date X" sections. The file should always reflect current state.
- If something is removed from the model or deferred, move it to the appropriate "Deferred / Out of Scope" section rather than deleting it entirely. Deferred decisions have context worth preserving.
- Keep formatting consistent with what is already in the file.

## General Development Guidelines

- Stack: React + TypeScript + Vite (frontend), Go (backend), PostgreSQL, S3-compatible storage (MinIO locally)
- Architecture: modular monolith, feature-first module structure
- SQL: explicit migrations, no heavy ORM magic
- The frontend feature structure mirrors the backend module structure
- Single-owner app for MVP — no multi-user collaboration
- Prefer explicit over clever; prefer simple over flexible until flexibility is earned by a real requirement

## Domain Notes

- This app tracks documents for citizenship-by-descent and genealogy-related cases
- Primary users are professionals collecting documents on behalf of applicants
- `recorded_*` fields on Document always reflect the original document in its original language — never populated from translations or apostilles
- `is_verified` on Document flips to false when an amendment is uploaded; stays true for apostille and translation uploads
- Spouse data lives on the marriage LifeEvent as flat fields — spouses are not Person records in the lineage graph
- PersonRelationship is parent-child only — no other relationship types
- ClaimLine is always present, even for single-LIRA cases (status: eligible)
