-- Document tracking: the details check records who made it, each phase records
-- when it entered its status and an optional reference, documents can be kept
-- out of submission, and three document types are added.

ALTER TYPE document_type ADD VALUE 'non_naturalization_certificate';
ALTER TYPE document_type ADD VALUE 'divorce_record';
ALTER TYPE document_type ADD VALUE 'court_order';

-- The check keeps its value and time. Who made earlier checks was never
-- recorded, so details_checked_by starts empty.
ALTER TABLE documents RENAME COLUMN is_verified TO details_checked;
ALTER TABLE documents RENAME COLUMN verified_at TO details_checked_at;
ALTER TABLE documents RENAME CONSTRAINT documents_is_verified_not_null TO documents_details_checked_not_null;
ALTER TABLE documents ADD COLUMN details_checked_by UUID REFERENCES users(id) ON DELETE SET NULL;
UPDATE documents SET details_checked_at = NULL WHERE NOT details_checked;
ALTER TABLE documents ADD CONSTRAINT document_check_recorded
    CHECK (details_checked OR (details_checked_at IS NULL AND details_checked_by IS NULL));

-- Each phase is dated from the document's last update, the best time known.
ALTER TABLE documents
    ADD COLUMN official_copy_status_since TIMESTAMPTZ,
    ADD COLUMN amendment_status_since     TIMESTAMPTZ,
    ADD COLUMN apostille_status_since     TIMESTAMPTZ,
    ADD COLUMN translation_status_since   TIMESTAMPTZ,
    ADD COLUMN official_copy_reference    TEXT,
    ADD COLUMN amendment_reference        TEXT,
    ADD COLUMN apostille_reference        TEXT,
    ADD COLUMN translation_reference      TEXT,
    ADD COLUMN for_submission             BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE documents SET
    official_copy_status_since = updated_at,
    amendment_status_since     = updated_at,
    apostille_status_since     = updated_at,
    translation_status_since   = updated_at;

ALTER TABLE documents
    ALTER COLUMN official_copy_status_since SET NOT NULL,
    ALTER COLUMN official_copy_status_since SET DEFAULT NOW(),
    ALTER COLUMN amendment_status_since     SET NOT NULL,
    ALTER COLUMN amendment_status_since     SET DEFAULT NOW(),
    ALTER COLUMN apostille_status_since     SET NOT NULL,
    ALTER COLUMN apostille_status_since     SET DEFAULT NOW(),
    ALTER COLUMN translation_status_since   SET NOT NULL,
    ALTER COLUMN translation_status_since   SET DEFAULT NOW();
