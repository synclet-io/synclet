-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE pipeline.bucket_size AS ENUM ('hourly', 'daily');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE pipeline.support_level AS ENUM ('community', 'certified', 'unknown');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE pipeline.source_type AS ENUM ('api', 'database', 'file', 'unknown');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
DO $$ BEGIN
    CREATE TYPE pipeline.release_stage AS ENUM ('generally_available', 'beta', 'alpha', 'custom', 'unknown');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
DECLARE
    col RECORD;
BEGIN
    FOR col IN
        SELECT c.table_name, c.column_name, t.enum_type, t.fallback
        FROM (VALUES
            ('repository_connectors', 'support_level', 'support_level', 'unknown'),
            ('repository_connectors', 'source_type', 'source_type', 'unknown'),
            ('repository_connectors', 'release_stage', 'release_stage', 'unknown'),
            ('stats_rollups', 'bucket_size', 'bucket_size', NULL)
        ) AS t(table_name, column_name, enum_type, fallback)
        JOIN information_schema.columns c
            ON c.table_schema = 'pipeline'
            AND c.table_name = t.table_name
            AND c.column_name = t.column_name
            AND c.data_type = 'text'
    LOOP
        EXECUTE format('ALTER TABLE pipeline.%I ALTER COLUMN %I DROP DEFAULT', col.table_name, col.column_name);
        IF col.fallback IS NULL THEN
            EXECUTE format(
                'ALTER TABLE pipeline.%I ALTER COLUMN %I TYPE pipeline.%I USING %I::pipeline.%I',
                col.table_name, col.column_name, col.enum_type, col.column_name, col.enum_type
            );
        ELSE
            EXECUTE format(
                'ALTER TABLE pipeline.%I ALTER COLUMN %I TYPE pipeline.%I USING (CASE WHEN %I = ANY (enum_range(NULL::pipeline.%I)::text[]) THEN %I ELSE %L END)::pipeline.%I',
                col.table_name, col.column_name, col.enum_type, col.column_name, col.enum_type, col.column_name, col.fallback, col.enum_type
            );
            EXECUTE format(
                'ALTER TABLE pipeline.%I ALTER COLUMN %I SET DEFAULT %L::pipeline.%I',
                col.table_name, col.column_name, col.fallback, col.enum_type
            );
        END IF;
    END LOOP;
END $$;

CREATE TABLE IF NOT EXISTS pipeline.workspace_settings (
    workspace_id UUID PRIMARY KEY,
    max_jobs_per_workspace INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline.stream_generations (
    connection_id UUID NOT NULL REFERENCES pipeline.connections(id) ON DELETE CASCADE,
    stream_namespace TEXT NOT NULL DEFAULT '',
    stream_name TEXT NOT NULL,
    generation_id BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (connection_id, stream_namespace, stream_name)
);
CREATE INDEX IF NOT EXISTS idx_stream_generations_connection_id ON pipeline.stream_generations(connection_id);

CREATE INDEX IF NOT EXISTS idx_jobs_status_created ON pipeline.jobs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_running_heartbeat ON pipeline.jobs(heartbeat_at) WHERE status = 'running';
CREATE INDEX IF NOT EXISTS idx_job_attempts_job_id ON pipeline.job_attempts(job_id);

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'catalog_discoveries_source_id_fkey'
            AND conrelid = 'pipeline.catalog_discoveries'::regclass
    ) THEN
        DELETE FROM pipeline.catalog_discoveries d
        WHERE NOT EXISTS (SELECT 1 FROM pipeline.sources s WHERE s.id = d.source_id);
        ALTER TABLE pipeline.catalog_discoveries
            ADD CONSTRAINT catalog_discoveries_source_id_fkey
            FOREIGN KEY (source_id) REFERENCES pipeline.sources(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'configured_catalogs_connection_id_fkey'
            AND conrelid = 'pipeline.configured_catalogs'::regclass
    ) THEN
        DELETE FROM pipeline.configured_catalogs cc
        WHERE NOT EXISTS (SELECT 1 FROM pipeline.connections c WHERE c.id = cc.connection_id);
        ALTER TABLE pipeline.configured_catalogs
            ADD CONSTRAINT configured_catalogs_connection_id_fkey
            FOREIGN KEY (connection_id) REFERENCES pipeline.connections(id) ON DELETE CASCADE;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
