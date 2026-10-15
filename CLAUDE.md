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
| `CLAUDE.md` (repo root) | This file. Instructions for AI-assisted development. |

### When to Update

- **domain-model.md** — whenever an entity is added, removed, or modified; whenever a field is added, renamed, or removed; whenever a domain rule is established or changed; whenever something is moved between MoSCoW tiers.
- **architecture.md** — whenever a stack decision is made or changed; whenever module boundaries are drawn or revised; whenever a new package or directory is added to the canonical structure.
- **api-conventions.md** — whenever an API endpoint is designed or changed; whenever a new convention is established.
- **milestones.md** — whenever a milestone is defined, revised, or completed; whenever a task within a milestone is checked off.

IMPORTANT: Changes to these documents, including this CLAUDE.md file, are decisions. Before editing one, propose the exact change and get explicit approval from the person you are working with, every time, even when you may edit other files freely. Approval covers only the change proposed. Propose changes as decisions are made rather than at the end of a session.

### How to Update

- The design docs (`domain-model.md`, `architecture.md`, `api-conventions.md`) describe the agreed design, which may run ahead of the code. `milestones.md` is the only place that says what has been built.
- Replace outdated content directly. These are not changelogs — do not append "as of date X" sections. The file should always reflect current state.
- If something is removed from the model or deferred, move it to the appropriate "Deferred / Out of Scope" section rather than deleting it entirely. Deferred decisions have context worth preserving.
- Keep formatting consistent with what is already in the file.

## General Development Guidelines

- Stack: React + TypeScript + Vite (frontend), Go (backend), PostgreSQL, S3-compatible storage (local development store to be chosen in Milestone 8)
- Architecture: modular monolith, feature-first module structure
- SQL: explicit migrations, no heavy ORM magic
- The frontend feature structure mirrors the backend module structure
- Multi-user with shared access for MVP: every user sees every case. Users are created with `make create-user`; there is no user-management UI and no per-user permissions
- The backend is the single authority for domain rules and derived values; the frontend presents them (see `architecture.md`, Division of Responsibility)
- Every write runs in one transaction together with its activity log entry (see `architecture.md`, Transactions)
- A module only writes its own tables (see `architecture.md`, Module Boundaries)
- Prefer explicit over clever; prefer simple over flexible until flexibility is earned by a real requirement

## Domain Notes

`docs/domain-model.md` is the authority on entities, fields, and domain rules. Read it before changing anything it covers.
