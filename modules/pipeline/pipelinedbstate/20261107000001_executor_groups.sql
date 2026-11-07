-- +goose Up
-- +goose StatementBegin
ALTER TABLE pipeline.executors ADD COLUMN IF NOT EXISTS executor_group TEXT NOT NULL DEFAULT 'default';
ALTER TABLE pipeline.sources ADD COLUMN IF NOT EXISTS executor_group TEXT NOT NULL DEFAULT 'default';
ALTER TABLE pipeline.destinations ADD COLUMN IF NOT EXISTS executor_group TEXT NOT NULL DEFAULT 'default';
ALTER TABLE pipeline.jobs ADD COLUMN IF NOT EXISTS executor_group TEXT NOT NULL DEFAULT 'default';
ALTER TABLE pipeline.jobs ADD COLUMN IF NOT EXISTS queue_reason TEXT;
ALTER TABLE pipeline.jobs ADD COLUMN IF NOT EXISTS stalled_at TIMESTAMPTZ;
ALTER TABLE pipeline.connector_tasks ADD COLUMN IF NOT EXISTS executor_group TEXT;
UPDATE pipeline.connector_tasks SET executor_group = 'default' WHERE task_type <> 'spec' AND executor_group IS NULL;
CREATE INDEX IF NOT EXISTS jobs_status_executor_group_idx ON pipeline.jobs (status, executor_group);
CREATE INDEX IF NOT EXISTS connector_tasks_status_executor_group_idx ON pipeline.connector_tasks (status, executor_group);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS pipeline.connector_tasks_status_executor_group_idx;
DROP INDEX IF EXISTS pipeline.jobs_status_executor_group_idx;
ALTER TABLE pipeline.connector_tasks DROP COLUMN IF EXISTS executor_group;
ALTER TABLE pipeline.jobs DROP COLUMN IF EXISTS stalled_at;
ALTER TABLE pipeline.jobs DROP COLUMN IF EXISTS queue_reason;
ALTER TABLE pipeline.jobs DROP COLUMN IF EXISTS executor_group;
ALTER TABLE pipeline.destinations DROP COLUMN IF EXISTS executor_group;
ALTER TABLE pipeline.sources DROP COLUMN IF EXISTS executor_group;
ALTER TABLE pipeline.executors DROP COLUMN IF EXISTS executor_group;
-- +goose StatementEnd
