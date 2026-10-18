-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE pipeline.executor_type AS ENUM ('docker', 'k8s');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS pipeline.executors (
    id UUID PRIMARY KEY,
    instance_id TEXT NOT NULL UNIQUE,
    type pipeline.executor_type NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    capacity INT NOT NULL DEFAULT 0,
    capabilities JSONB NOT NULL DEFAULT '{}',
    resource_defaults JSONB NOT NULL DEFAULT '{}',
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pipeline.executors;
DROP TYPE IF EXISTS pipeline.executor_type;
-- +goose StatementEnd
