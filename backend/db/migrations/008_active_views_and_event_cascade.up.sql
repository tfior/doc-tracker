-- A document's person is its life event's person. Earlier code could break
-- this when a life event or document was reassigned; repair any such rows.
UPDATE documents d
SET person_id = le.person_id, updated_at = NOW()
FROM life_events le
WHERE d.life_event_id = le.id AND d.person_id <> le.person_id;

-- Active views: one definition, per entity, of "active", meaning not in the
-- trash and not beneath anything in the trash (domain model, Soft-delete and
-- Trash). Each view is owned by the module that owns the entity. Lists read
-- from them, and active checks lock rows through them with FOR SHARE, which
-- locks the entity and every parent it depends on.
--
-- The views expose only identifying columns, so adding a column to a table
-- never requires recreating a view. Queries join back to the table for data.

-- cases module
CREATE VIEW active_cases AS
    SELECT c.id
    FROM cases c
    WHERE c.deleted_at IS NULL;

-- people module
CREATE VIEW active_people AS
    SELECT p.id, p.case_id
    FROM people p
    JOIN cases c ON c.id = p.case_id
    WHERE p.deleted_at IS NULL AND c.deleted_at IS NULL;

-- lifeevents module
CREATE VIEW active_life_events AS
    SELECT le.id, le.case_id, le.person_id
    FROM life_events le
    JOIN active_people p ON p.id = le.person_id
    WHERE le.deleted_at IS NULL;

-- documents module. A document under a life event is active only while that
-- event is.
CREATE VIEW active_documents AS
    SELECT d.id, d.case_id, d.person_id, d.life_event_id
    FROM documents d
    JOIN active_people p ON p.id = d.person_id
    WHERE d.deleted_at IS NULL
      AND (d.life_event_id IS NULL
           OR EXISTS (SELECT 1 FROM active_life_events le WHERE le.id = d.life_event_id));

-- claimlines module
CREATE VIEW active_claim_lines AS
    SELECT cl.id, cl.case_id, cl.root_person_id
    FROM claim_lines cl
    JOIN active_people p ON p.id = cl.root_person_id
    WHERE cl.deleted_at IS NULL;

-- Permanently deleting a life event deletes its documents (domain model,
-- Soft-delete and Trash). Previously they survived, detached from any event.
ALTER TABLE documents
    DROP CONSTRAINT documents_life_event_id_fkey,
    ADD CONSTRAINT documents_life_event_id_fkey
        FOREIGN KEY (life_event_id) REFERENCES life_events(id) ON DELETE CASCADE;
