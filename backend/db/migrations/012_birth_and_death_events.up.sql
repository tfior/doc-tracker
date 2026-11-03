-- A person's birth and death live on their birth and death life events, and a
-- person has at most one of each (domain-model.md, Person and rule 6). The
-- person's own birth_date, birth_place and death_date move to those events.
-- When both have a value and they differ, the event keeps its own and the
-- person's is written into the event's notes, so nothing is lost.

-- Which of two birth events is the real one is a judgement, so a person with
-- more than one active birth or death event stops the migration.
DO $$
DECLARE dup record;
BEGIN
    SELECT p.first_name || ' ' || p.last_name AS name, le.event_type
    INTO dup
    FROM life_events le JOIN people p ON p.id = le.person_id
    WHERE le.event_type IN ('birth', 'death') AND le.deleted_at IS NULL
    GROUP BY p.id, p.first_name, p.last_name, le.event_type
    HAVING COUNT(*) > 1
    LIMIT 1;
    IF FOUND THEN
        RAISE EXCEPTION '% has more than one % life event. Delete or change all but one, then run the migrations again.',
            dup.name, dup.event_type;
    END IF;
END $$;

-- Birth: fill the empty fields of an existing event, noting disagreements.
UPDATE life_events le SET
    event_date           = COALESCE(le.event_date, to_char(p.birth_date, 'YYYY-MM-DD')),
    event_date_qualifier = CASE WHEN le.event_date IS NULL AND p.birth_date IS NOT NULL
                                THEN 'exact' ELSE le.event_date_qualifier END,
    event_place          = COALESCE(le.event_place, p.birth_place),
    notes = NULLIF(concat_ws(E'\n', le.notes,
        CASE WHEN le.event_date <> to_char(p.birth_date, 'YYYY-MM-DD')
             THEN 'Birth date on the person record: ' || to_char(p.birth_date, 'YYYY-MM-DD') END,
        CASE WHEN le.event_place <> p.birth_place
             THEN 'Birth place on the person record: ' || p.birth_place END), '')
FROM people p
WHERE p.id = le.person_id AND le.event_type = 'birth' AND le.deleted_at IS NULL
  AND (p.birth_date IS NOT NULL OR p.birth_place IS NOT NULL);

-- Birth: create the event for everyone else with a birth date or place.
INSERT INTO life_events (case_id, person_id, event_type, event_date, event_date_qualifier, event_place)
SELECT p.case_id, p.id, 'birth', to_char(p.birth_date, 'YYYY-MM-DD'),
       CASE WHEN p.birth_date IS NOT NULL THEN 'exact'::date_qualifier END, p.birth_place
FROM people p
WHERE (p.birth_date IS NOT NULL OR p.birth_place IS NOT NULL)
  AND NOT EXISTS (SELECT 1 FROM life_events le
                  WHERE le.person_id = p.id AND le.event_type = 'birth' AND le.deleted_at IS NULL);

-- Death: the same, for the date alone.
UPDATE life_events le SET
    event_date           = COALESCE(le.event_date, to_char(p.death_date, 'YYYY-MM-DD')),
    event_date_qualifier = CASE WHEN le.event_date IS NULL THEN 'exact' ELSE le.event_date_qualifier END,
    notes = NULLIF(concat_ws(E'\n', le.notes,
        CASE WHEN le.event_date <> to_char(p.death_date, 'YYYY-MM-DD')
             THEN 'Death date on the person record: ' || to_char(p.death_date, 'YYYY-MM-DD') END), '')
FROM people p
WHERE p.id = le.person_id AND le.event_type = 'death' AND le.deleted_at IS NULL
  AND p.death_date IS NOT NULL;

INSERT INTO life_events (case_id, person_id, event_type, event_date, event_date_qualifier)
SELECT p.case_id, p.id, 'death', to_char(p.death_date, 'YYYY-MM-DD'), 'exact'
FROM people p
WHERE p.death_date IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM life_events le
                  WHERE le.person_id = p.id AND le.event_type = 'death' AND le.deleted_at IS NULL);

ALTER TABLE people DROP COLUMN birth_date, DROP COLUMN birth_place, DROP COLUMN death_date;

CREATE UNIQUE INDEX life_events_one_birth_one_death ON life_events (person_id, event_type)
    WHERE event_type IN ('birth', 'death') AND deleted_at IS NULL;
