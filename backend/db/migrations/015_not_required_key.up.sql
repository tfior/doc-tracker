-- "Not Required" on the official copy phase means a document is not needed
-- (domain rule 29). A system key lets the backend recognise it without
-- relying on its label. Readiness depends on it, so the migration fails
-- rather than leave the key unset.

DO $$
DECLARE
    changed integer;
BEGIN
    UPDATE document_statuses SET system_key = 'not_required'
    WHERE is_system AND phase = 'any' AND label = 'Not Required';
    GET DIAGNOSTICS changed = ROW_COUNT;
    IF changed <> 1 THEN
        RAISE EXCEPTION 'expected one system "Not Required" status, found %', changed;
    END IF;
END $$;
