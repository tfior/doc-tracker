-- Replace the claim_line_status enum with research-workflow-appropriate values.
--
-- Old → New mapping:
--   active     → researching
--   suspended  → paused
--   eliminated → ineligible
--   confirmed  → eligible
--
-- 'not_yet_researched' is a new value with no old equivalent.

CREATE TYPE claim_line_status_new AS ENUM (
    'not_yet_researched',
    'researching',
    'paused',
    'ineligible',
    'eligible'
);

ALTER TABLE claim_lines ALTER COLUMN status DROP DEFAULT;

ALTER TABLE claim_lines
    ALTER COLUMN status TYPE claim_line_status_new
    USING CASE status::text
        WHEN 'active'     THEN 'researching'::claim_line_status_new
        WHEN 'suspended'  THEN 'paused'::claim_line_status_new
        WHEN 'eliminated' THEN 'ineligible'::claim_line_status_new
        WHEN 'confirmed'  THEN 'eligible'::claim_line_status_new
    END;

DROP TYPE claim_line_status;
ALTER TYPE claim_line_status_new RENAME TO claim_line_status;

ALTER TABLE claim_lines ALTER COLUMN status SET DEFAULT 'not_yet_researched'::claim_line_status;
