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

CREATE INDEX ON activity_logs(case_id);
CREATE INDEX ON activity_logs(user_id);
CREATE INDEX ON activity_logs(created_at);
