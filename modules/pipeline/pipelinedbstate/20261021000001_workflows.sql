-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE pipeline.workflow_run_status AS ENUM ('running', 'succeeded', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE pipeline.workflow_run_trigger AS ENUM ('schedule', 'manual');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE pipeline.workflow_step_run_status AS ENUM ('pending', 'queued', 'succeeded', 'failed', 'skipped');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS pipeline.workflows (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    schedule TEXT NULL,
    next_scheduled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, name)
);
CREATE INDEX IF NOT EXISTS workflows_next_scheduled_at_idx ON pipeline.workflows (next_scheduled_at) WHERE next_scheduled_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS pipeline.workflow_steps (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES pipeline.workflows (id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES pipeline.connections (id) ON DELETE RESTRICT,
    UNIQUE (workflow_id, connection_id)
);
CREATE INDEX IF NOT EXISTS workflow_steps_connection_id_idx ON pipeline.workflow_steps (connection_id);

CREATE TABLE IF NOT EXISTS pipeline.workflow_step_dependencies (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES pipeline.workflows (id) ON DELETE CASCADE,
    step_id UUID NOT NULL REFERENCES pipeline.workflow_steps (id) ON DELETE CASCADE,
    upstream_step_id UUID NOT NULL REFERENCES pipeline.workflow_steps (id) ON DELETE CASCADE,
    UNIQUE (step_id, upstream_step_id)
);
CREATE INDEX IF NOT EXISTS workflow_step_dependencies_workflow_id_idx ON pipeline.workflow_step_dependencies (workflow_id);

CREATE TABLE IF NOT EXISTS pipeline.workflow_runs (
    id UUID PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES pipeline.workflows (id) ON DELETE CASCADE,
    status pipeline.workflow_run_status NOT NULL,
    trigger pipeline.workflow_run_trigger NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS workflow_runs_workflow_id_idx ON pipeline.workflow_runs (workflow_id, started_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS workflow_runs_one_running_idx ON pipeline.workflow_runs (workflow_id) WHERE status = 'running';

CREATE TABLE IF NOT EXISTS pipeline.workflow_step_runs (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES pipeline.workflow_runs (id) ON DELETE CASCADE,
    step_id UUID NOT NULL,
    connection_id UUID NOT NULL,
    status pipeline.workflow_step_run_status NOT NULL,
    job_id UUID NULL,
    error TEXT NULL,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS workflow_step_runs_run_id_idx ON pipeline.workflow_step_runs (run_id);
CREATE INDEX IF NOT EXISTS workflow_step_runs_connection_status_idx ON pipeline.workflow_step_runs (connection_id, status);

ALTER TABLE pipeline.jobs ADD COLUMN IF NOT EXISTS workflow_step_run_id UUID NULL
    REFERENCES pipeline.workflow_step_runs (id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS jobs_workflow_step_run_id_idx ON pipeline.jobs (workflow_step_run_id) WHERE workflow_step_run_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS pipeline.jobs_workflow_step_run_id_idx;
ALTER TABLE pipeline.jobs DROP COLUMN IF EXISTS workflow_step_run_id;
DROP TABLE IF EXISTS pipeline.workflow_step_runs;
DROP TABLE IF EXISTS pipeline.workflow_runs;
DROP TABLE IF EXISTS pipeline.workflow_step_dependencies;
DROP TABLE IF EXISTS pipeline.workflow_steps;
DROP TABLE IF EXISTS pipeline.workflows;
DROP TYPE IF EXISTS pipeline.workflow_step_run_status;
DROP TYPE IF EXISTS pipeline.workflow_run_trigger;
DROP TYPE IF EXISTS pipeline.workflow_run_status;
-- +goose StatementEnd
