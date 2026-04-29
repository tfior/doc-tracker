-- Update activity_logs.case_id FK to CASCADE so permanently deleting a case
-- also removes its audit trail. The alternative (blocking the delete) is not
-- useful since a permanently-deleted case has no context for its logs.

ALTER TABLE activity_logs
    DROP CONSTRAINT activity_logs_case_id_fkey,
    ADD CONSTRAINT activity_logs_case_id_fkey
        FOREIGN KEY (case_id) REFERENCES cases(id) ON DELETE CASCADE;
