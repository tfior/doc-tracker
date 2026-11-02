-- Every editable record carries a version that rises with each change, so a
-- save based on an out-of-date copy can be refused (api-conventions.md,
-- Concurrent Edits).
ALTER TABLE cases       ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE people      ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE life_events ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE documents   ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE claim_lines ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
