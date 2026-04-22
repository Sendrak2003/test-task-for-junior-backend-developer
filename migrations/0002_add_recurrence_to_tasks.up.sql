ALTER TABLE tasks ADD COLUMN IF NOT EXISTS recurrence JSONB;

-- Индекс для фильтрации задач по типу периодичности
CREATE INDEX IF NOT EXISTS idx_tasks_recurrence_type
    ON tasks ((recurrence->>'type'))
    WHERE recurrence IS NOT NULL;
