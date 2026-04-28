-- Current schema snapshot. This file is for reference only — do not run it directly.
-- Authoritative schema is defined by the migration files in db/migrations/.

-- Enums

CREATE TYPE activity_action AS ENUM ('created', 'updated', 'deleted');

CREATE TYPE activity_entity_type AS ENUM (
    'case',
    'person',
    'person_relationship',
    'life_event',
    'document',
    'file_attachment',
    'claim_line'
);

CREATE TYPE case_status AS ENUM ('active', 'archived', 'complete');

CREATE TYPE claim_line_status AS ENUM ('active', 'suspended', 'eliminated', 'confirmed');

CREATE TYPE life_event_type AS ENUM ('birth', 'marriage', 'death', 'naturalization', 'immigration', 'other');

CREATE TYPE document_type AS ENUM ('birth_certificate', 'marriage_certificate', 'naturalization', 'death_certificate', 'other');

CREATE TYPE attachment_type AS ENUM ('original', 'apostille', 'translation', 'amendment');

CREATE TYPE progress_bucket AS ENUM ('not_started', 'in_progress', 'complete');

CREATE TYPE document_system_key AS ENUM ('pending', 'collected', 'verified', 'unobtainable');


-- Tables

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    first_name    TEXT        NOT NULL,
    last_name     TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE cases (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title                  TEXT        NOT NULL,
    status                 case_status NOT NULL DEFAULT 'active',
    primary_root_person_id UUID        REFERENCES people(id) ON DELETE SET NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at             TIMESTAMPTZ
);

CREATE TABLE people (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id     UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    first_name  TEXT        NOT NULL,
    last_name   TEXT        NOT NULL,
    birth_date  DATE,
    birth_place TEXT,
    death_date  DATE,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE person_relationships (
    person_id UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    parent_id UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    case_id   UUID NOT NULL REFERENCES cases(id)  ON DELETE CASCADE,
    PRIMARY KEY (person_id, parent_id)
);

CREATE TABLE claim_lines (
    id             UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id        UUID              NOT NULL REFERENCES cases(id)  ON DELETE CASCADE,
    root_person_id UUID              NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    status         claim_line_status NOT NULL DEFAULT 'active',
    notes          TEXT,
    created_at     TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE TABLE life_events (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id            UUID            NOT NULL REFERENCES cases(id)  ON DELETE CASCADE,
    person_id          UUID            NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    event_type         life_event_type NOT NULL,
    event_date         DATE,
    event_place        TEXT,
    spouse_name        TEXT,
    spouse_birth_date  DATE,
    spouse_birth_place TEXT,
    notes              TEXT,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

CREATE TABLE document_statuses (
    id              UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    label           TEXT                NOT NULL,
    is_system       BOOLEAN             NOT NULL DEFAULT FALSE,
    system_key      document_system_key UNIQUE,
    progress_bucket progress_bucket     NOT NULL,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

CREATE TABLE documents (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id              UUID          NOT NULL REFERENCES cases(id)       ON DELETE CASCADE,
    person_id            UUID          NOT NULL REFERENCES people(id)      ON DELETE CASCADE,
    life_event_id        UUID          REFERENCES life_events(id)          ON DELETE SET NULL,
    status_id            UUID          NOT NULL REFERENCES document_statuses(id),
    document_type        document_type NOT NULL,
    title                TEXT          NOT NULL,
    issuing_authority    TEXT,
    issue_date           DATE,
    recorded_date        DATE,
    recorded_given_name  TEXT,
    recorded_surname     TEXT,
    recorded_birth_date  DATE,
    recorded_birth_place TEXT,
    is_verified          BOOLEAN       NOT NULL DEFAULT FALSE,
    verified_at          TIMESTAMPTZ,
    notes                TEXT,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE TABLE file_attachments (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID            NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    storage_key     TEXT            NOT NULL,
    filename        TEXT            NOT NULL,
    content_type    TEXT            NOT NULL,
    size_bytes      INTEGER         NOT NULL,
    is_canonical    BOOLEAN         NOT NULL DEFAULT FALSE,
    attachment_type attachment_type NOT NULL,
    superseded_at   TIMESTAMPTZ,
    notes           TEXT,
    uploaded_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE activity_logs (
    id          UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id     UUID                 NOT NULL REFERENCES cases(id),
    user_id     UUID                 NOT NULL REFERENCES users(id),
    action      activity_action      NOT NULL,
    entity_type activity_entity_type NOT NULL,
    entity_id   UUID                 NOT NULL,
    entity_name TEXT                 NOT NULL,
    changes     JSONB,
    created_at  TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);
