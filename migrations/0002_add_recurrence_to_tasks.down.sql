DROP INDEX IF EXISTS idx_tasks_recurrence_type;

ALTER TABLE tasks DROP COLUMN IF EXISTS recurrence;
