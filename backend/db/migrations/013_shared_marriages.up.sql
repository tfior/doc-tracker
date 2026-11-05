-- A marriage between two people in a case is one event, linked to the spouse
-- through spouse_person_id, and a divorce is recorded on the marriage it ended.
-- Spouse and divorce details belong to marriages only (domain-model.md,
-- Marriages and Spouses).

ALTER TABLE life_events
    ADD COLUMN spouse_person_id uuid REFERENCES people(id) ON DELETE SET NULL,
    ADD COLUMN divorce_date text,
    ADD COLUMN divorce_date_qualifier date_qualifier,
    ADD COLUMN divorce_date_end text,
    ADD COLUMN divorce_place text,
    ADD CONSTRAINT divorce_date_form CHECK (is_partial_date(divorce_date) AND is_partial_date(divorce_date_end)),
    ADD CONSTRAINT divorce_date_qualified CHECK ((divorce_date IS NULL) = (divorce_date_qualifier IS NULL)),
    ADD CONSTRAINT divorce_date_range CHECK ((divorce_date_end IS NOT NULL) = COALESCE(divorce_date_qualifier = 'between', false));

CREATE INDEX life_events_spouse_person_id ON life_events (spouse_person_id);

-- Spouse details saved on events that are not marriages move into the event's
-- notes, so nothing is lost.
UPDATE life_events SET
    notes = NULLIF(concat_ws(E'\n', notes,
        'Spouse name: ' || spouse_name,
        'Spouse birth date: ' ||
            CASE spouse_birth_date_qualifier
                WHEN 'exact' THEN spouse_birth_date
                WHEN 'between' THEN 'between ' || spouse_birth_date || ' and ' || spouse_birth_date_end
                ELSE spouse_birth_date_qualifier::text || ' ' || spouse_birth_date
            END,
        'Spouse birth place: ' || spouse_birth_place), ''),
    spouse_name = NULL,
    spouse_birth_date = NULL, spouse_birth_date_qualifier = NULL, spouse_birth_date_end = NULL,
    spouse_birth_place = NULL
WHERE event_type <> 'marriage'
  AND (spouse_name IS NOT NULL OR spouse_birth_date IS NOT NULL OR spouse_birth_place IS NOT NULL);

ALTER TABLE life_events
    ADD CONSTRAINT spouse_and_divorce_on_marriages CHECK (event_type = 'marriage' OR (
        spouse_person_id IS NULL AND spouse_name IS NULL AND spouse_birth_date IS NULL
        AND spouse_birth_place IS NULL AND divorce_date IS NULL AND divorce_place IS NULL)),
    ADD CONSTRAINT linked_spouse_or_flat_fields CHECK (spouse_person_id IS NULL OR (
        spouse_name IS NULL AND spouse_birth_date IS NULL AND spouse_birth_place IS NULL)),
    ADD CONSTRAINT spouse_is_another_person CHECK (spouse_person_id <> person_id);

-- A marriage with a linked spouse is active while either spouse is: trashing
-- one leaves it under the other (rule 26). A document under a life event is
-- active while its event is, so a shared marriage's documents stay with it.
-- The views no longer join the people they check, so their active checks lock
-- those people themselves.
CREATE OR REPLACE VIEW active_life_events AS
    SELECT le.id, le.case_id, le.person_id
    FROM life_events le
    JOIN active_cases c ON c.id = le.case_id
    WHERE le.deleted_at IS NULL
      AND (EXISTS (SELECT 1 FROM active_people p WHERE p.id = le.person_id)
           OR EXISTS (SELECT 1 FROM active_people p WHERE p.id = le.spouse_person_id));

CREATE OR REPLACE VIEW active_documents AS
    SELECT d.id, d.case_id, d.person_id, d.life_event_id
    FROM documents d
    JOIN active_cases c ON c.id = d.case_id
    WHERE d.deleted_at IS NULL
      AND CASE WHEN d.life_event_id IS NULL
               THEN EXISTS (SELECT 1 FROM active_people p WHERE p.id = d.person_id)
               ELSE EXISTS (SELECT 1 FROM active_life_events le WHERE le.id = d.life_event_id)
          END;
