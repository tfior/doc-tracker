-- Enums

CREATE TYPE case_status AS ENUM ('active', 'archived', 'complete');

CREATE TYPE claim_line_status AS ENUM ('active', 'suspended', 'eliminated', 'confirmed');

CREATE TYPE life_event_type AS ENUM ('birth', 'marriage', 'death', 'naturalization', 'immigration', 'other');

CREATE TYPE document_type AS ENUM ('birth_certificate', 'marriage_certificate', 'naturalization', 'death_certificate', 'other');

CREATE TYPE attachment_type AS ENUM ('original', 'apostille', 'translation', 'amendment');

CREATE TYPE progress_bucket AS ENUM ('not_started', 'in_progress', 'complete');

CREATE TYPE document_system_key AS ENUM ('pending', 'collected', 'verified', 'unobtainable');


-- cases
-- primary_root_person_id FK is added after people to resolve the circular dependency.

CREATE TABLE cases (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title                  TEXT        NOT NULL,
    status                 case_status NOT NULL DEFAULT 'active',
    primary_root_person_id UUID,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- people

CREATE TABLE people (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id     UUID        NOT NULL REFERENCES cases(id),
    first_name  TEXT        NOT NULL,
    last_name   TEXT        NOT NULL,
    birth_date  DATE,
    birth_place TEXT,
    death_date  DATE,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE cases
    ADD CONSTRAINT cases_primary_root_person_id_fkey
    FOREIGN KEY (primary_root_person_id) REFERENCES people(id);


-- person_relationships

CREATE TABLE person_relationships (
    person_id UUID NOT NULL REFERENCES people(id),
    parent_id UUID NOT NULL REFERENCES people(id),
    case_id   UUID NOT NULL REFERENCES cases(id),
    PRIMARY KEY (person_id, parent_id)
);


-- claim_lines

CREATE TABLE claim_lines (
    id             UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id        UUID              NOT NULL REFERENCES cases(id),
    root_person_id UUID              NOT NULL REFERENCES people(id),
    status         claim_line_status NOT NULL DEFAULT 'active',
    notes          TEXT,
    created_at     TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);


-- life_events

CREATE TABLE life_events (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id            UUID            NOT NULL REFERENCES cases(id),
    person_id          UUID            NOT NULL REFERENCES people(id),
    event_type         life_event_type NOT NULL,
    event_date         DATE,
    event_place        TEXT,
    spouse_name        TEXT,
    spouse_birth_date  DATE,
    spouse_birth_place TEXT,
    notes              TEXT,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);


-- document_statuses
-- system_key has a unique constraint; NULL values (user-defined statuses) are exempt
-- because NULL != NULL in SQL.

CREATE TABLE document_statuses (
    id              UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    label           TEXT                NOT NULL,
    is_system       BOOLEAN             NOT NULL DEFAULT FALSE,
    system_key      document_system_key UNIQUE,
    progress_bucket progress_bucket     NOT NULL,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

INSERT INTO document_statuses (label, is_system, system_key, progress_bucket) VALUES
    ('Pending',      TRUE, 'pending',      'not_started'),
    ('Collected',    TRUE, 'collected',    'in_progress'),
    ('Verified',     TRUE, 'verified',     'complete'),
    ('Unobtainable', TRUE, 'unobtainable', 'complete');


-- documents

CREATE TABLE documents (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id              UUID          NOT NULL REFERENCES cases(id),
    person_id            UUID          NOT NULL REFERENCES people(id),
    life_event_id        UUID          REFERENCES life_events(id),
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
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);


-- file_attachments

CREATE TABLE file_attachments (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID            NOT NULL REFERENCES documents(id),
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


-- Indexes on foreign keys
-- PostgreSQL does not automatically index FK columns.

CREATE INDEX ON people(case_id);
CREATE INDEX ON person_relationships(person_id);
CREATE INDEX ON person_relationships(parent_id);
CREATE INDEX ON person_relationships(case_id);
CREATE INDEX ON claim_lines(case_id);
CREATE INDEX ON life_events(case_id);
CREATE INDEX ON life_events(person_id);
CREATE INDEX ON documents(case_id);
CREATE INDEX ON documents(person_id);
CREATE INDEX ON documents(life_event_id);
CREATE INDEX ON documents(status_id);
CREATE INDEX ON file_attachments(document_id);
