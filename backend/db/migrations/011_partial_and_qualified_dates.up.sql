-- Dates that describe people, events and documents may be known only in part
-- (domain-model.md, Field Types and Dates). A partial date is text: "1890",
-- "1890-03" or "1890-03-15". The application checks that the date exists; the
-- database checks its form. Life event dates are qualified dates, stored as
-- three columns: the date, its qualifier, and for "between" the range's end.
-- Dates already entered were full dates, so they carry over as exact.

CREATE FUNCTION is_partial_date(d TEXT) RETURNS BOOLEAN
    LANGUAGE sql IMMUTABLE
    RETURN d ~ '^[0-9]{4}(-(0[1-9]|1[0-2])(-(0[1-9]|[12][0-9]|3[01]))?)?$';

CREATE TYPE date_qualifier AS ENUM ('exact', 'about', 'before', 'after', 'between');

ALTER TABLE life_events
    ALTER COLUMN event_date        TYPE TEXT USING to_char(event_date, 'YYYY-MM-DD'),
    ALTER COLUMN spouse_birth_date TYPE TEXT USING to_char(spouse_birth_date, 'YYYY-MM-DD'),
    ADD COLUMN event_date_qualifier        date_qualifier,
    ADD COLUMN event_date_end              TEXT,
    ADD COLUMN spouse_birth_date_qualifier date_qualifier,
    ADD COLUMN spouse_birth_date_end       TEXT;

UPDATE life_events SET event_date_qualifier = 'exact' WHERE event_date IS NOT NULL;
UPDATE life_events SET spouse_birth_date_qualifier = 'exact' WHERE spouse_birth_date IS NOT NULL;

ALTER TABLE life_events
    ADD CONSTRAINT event_date_form CHECK (is_partial_date(event_date) AND is_partial_date(event_date_end)),
    ADD CONSTRAINT event_date_qualified CHECK ((event_date IS NULL) = (event_date_qualifier IS NULL)),
    ADD CONSTRAINT event_date_range CHECK (
        (event_date_end IS NOT NULL) = COALESCE(event_date_qualifier = 'between', FALSE)),
    ADD CONSTRAINT spouse_birth_date_form CHECK (
        is_partial_date(spouse_birth_date) AND is_partial_date(spouse_birth_date_end)),
    ADD CONSTRAINT spouse_birth_date_qualified CHECK (
        (spouse_birth_date IS NULL) = (spouse_birth_date_qualifier IS NULL)),
    ADD CONSTRAINT spouse_birth_date_range CHECK (
        (spouse_birth_date_end IS NOT NULL) = COALESCE(spouse_birth_date_qualifier = 'between', FALSE));

-- Document dates transcribe the document, so they are partial dates with no
-- qualifier. recorded_date was always the date of registration.
ALTER TABLE documents RENAME COLUMN recorded_date TO registration_date;

ALTER TABLE documents
    ALTER COLUMN issue_date          TYPE TEXT USING to_char(issue_date, 'YYYY-MM-DD'),
    ALTER COLUMN registration_date   TYPE TEXT USING to_char(registration_date, 'YYYY-MM-DD'),
    ALTER COLUMN recorded_birth_date TYPE TEXT USING to_char(recorded_birth_date, 'YYYY-MM-DD'),
    ADD CONSTRAINT document_dates_form CHECK (
        is_partial_date(issue_date) AND is_partial_date(registration_date) AND is_partial_date(recorded_birth_date));
