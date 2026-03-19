-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS pipeline;

-- Enum types
CREATE TYPE pipeline.connection_status AS ENUM ('active', 'inactive', 'paused');
CREATE TYPE pipeline.schema_change_policy AS ENUM ('propagate', 'ignore', 'pause');
CREATE TYPE pipeline.job_status AS ENUM ('scheduled', 'starting', 'running', 'completed', 'failed', 'cancelled');
CREATE TYPE pipeline.job_type AS ENUM ('sync', 'discover', 'check');
CREATE TYPE pipeline.namespace_definition AS ENUM ('source', 'destination', 'custom');
CREATE TYPE pipeline.state_type AS ENUM ('stream', 'global', 'legacy');
CREATE TYPE pipeline.bucket_size AS ENUM ('hourly', 'daily');
CREATE TYPE pipeline.connector_type AS ENUM ('source', 'destination');
CREATE TYPE pipeline.connector_task_type AS ENUM ('check', 'spec', 'discover');
CREATE TYPE pipeline.connector_task_status AS ENUM ('pending', 'running', 'completed', 'failed');
CREATE TYPE pipeline.repository_status AS ENUM ('syncing', 'synced', 'failed');
CREATE TYPE pipeline.support_level AS ENUM ('community', 'certified', 'unknown');
CREATE TYPE pipeline.source_type AS ENUM ('api', 'database', 'file', 'unknown');
CREATE TYPE pipeline.release_stage AS ENUM ('generally_available', 'beta', 'alpha', 'custom', 'unknown');

-- Repositories
CREATE TABLE pipeline.repositories (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    auth_header TEXT,
    status pipeline.repository_status NOT NULL DEFAULT 'syncing',
    last_synced_at TIMESTAMPTZ,
    connector_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_repositories_workspace_id ON pipeline.repositories(workspace_id);

-- Repository Connectors
CREATE TABLE pipeline.repository_connectors (
    id UUID PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES pipeline.repositories(id) ON DELETE CASCADE,
    docker_repository TEXT NOT NULL,
    docker_image_tag TEXT NOT NULL,
    name TEXT NOT NULL,
    connector_type pipeline.connector_type NOT NULL,
    documentation_url TEXT NOT NULL DEFAULT '',
    release_stage pipeline.release_stage NOT NULL DEFAULT 'unknown',
    icon_url TEXT NOT NULL DEFAULT '',
    spec JSONB NOT NULL DEFAULT '{}',
    support_level pipeline.support_level NOT NULL DEFAULT 'unknown',
    license TEXT NOT NULL DEFAULT '',
    source_type pipeline.source_type NOT NULL DEFAULT 'unknown',
    metadata JSONB NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_repo_connectors_repository_id ON pipeline.repository_connectors(repository_id);
CREATE INDEX idx_repo_connectors_type ON pipeline.repository_connectors(connector_type);

-- Managed Connectors
CREATE TABLE pipeline.managed_connectors (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    docker_image TEXT NOT NULL,
    docker_tag TEXT NOT NULL,
    name TEXT NOT NULL,
    connector_type pipeline.connector_type NOT NULL,
    spec JSONB,
    repository_id UUID REFERENCES pipeline.repositories(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_managed_connectors_workspace_id ON pipeline.managed_connectors(workspace_id);

-- Sources
CREATE TABLE pipeline.sources (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    managed_connector_id UUID NOT NULL REFERENCES pipeline.managed_connectors(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    runtime_config JSONB
);
CREATE INDEX idx_sources_workspace_id ON pipeline.sources(workspace_id);
CREATE INDEX idx_sources_managed_connector_id ON pipeline.sources(managed_connector_id);

-- Destinations
CREATE TABLE pipeline.destinations (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    managed_connector_id UUID NOT NULL REFERENCES pipeline.managed_connectors(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    runtime_config JSONB
);
CREATE INDEX idx_destinations_workspace_id ON pipeline.destinations(workspace_id);
CREATE INDEX idx_destinations_managed_connector_id ON pipeline.destinations(managed_connector_id);

-- Connections
CREATE TABLE pipeline.connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    status pipeline.connection_status NOT NULL DEFAULT 'active',
    source_id UUID NOT NULL,
    destination_id UUID NOT NULL,
    schedule TEXT,
    schema_change_policy pipeline.schema_change_policy NOT NULL DEFAULT 'pause',
    max_attempts INT NOT NULL DEFAULT 3,
    namespace_definition pipeline.namespace_definition NOT NULL DEFAULT 'source',
    custom_namespace_format TEXT,
    stream_prefix TEXT,
    next_scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_connections_workspace_id ON pipeline.connections(workspace_id);
CREATE INDEX idx_connections_source_id ON pipeline.connections(source_id);
CREATE INDEX idx_connections_destination_id ON pipeline.connections(destination_id);
CREATE INDEX idx_connections_next_scheduled_at ON pipeline.connections(next_scheduled_at) WHERE next_scheduled_at IS NOT NULL AND status = 'active';

-- Jobs
CREATE TABLE pipeline.jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES pipeline.connections(id) ON DELETE CASCADE,
    status pipeline.job_status NOT NULL DEFAULT 'scheduled',
    job_type pipeline.job_type NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error TEXT,
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    worker_id TEXT,
    heartbeat_at TIMESTAMPTZ,
    k8s_job_name TEXT,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_jobs_connection_id ON pipeline.jobs(connection_id);
CREATE INDEX idx_jobs_status_scheduled ON pipeline.jobs(status, scheduled_at) WHERE status = 'scheduled';
CREATE INDEX idx_jobs_status_created ON pipeline.jobs(status, created_at);
CREATE UNIQUE INDEX idx_jobs_one_active_per_connection ON pipeline.jobs(connection_id) WHERE status IN ('scheduled', 'starting', 'running');
CREATE INDEX idx_jobs_running_heartbeat ON pipeline.jobs(heartbeat_at) WHERE status = 'running';

-- Job Attempts
CREATE TABLE pipeline.job_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES pipeline.jobs(id) ON DELETE CASCADE,
    attempt_number INT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error TEXT,
    sync_stats_json JSONB NOT NULL DEFAULT '{}',
    UNIQUE(job_id, attempt_number)
);
CREATE INDEX idx_job_attempts_job_id ON pipeline.job_attempts(job_id);

-- Catalog Discoveries
CREATE TABLE pipeline.catalog_discoveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES pipeline.sources(id) ON DELETE CASCADE,
    version INT NOT NULL,
    catalog_json JSONB NOT NULL,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source_id, version)
);
CREATE INDEX idx_catalog_discoveries_source_id ON pipeline.catalog_discoveries(source_id);

-- Configured Catalogs
CREATE TABLE pipeline.configured_catalogs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL UNIQUE REFERENCES pipeline.connections(id) ON DELETE CASCADE,
    streams_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Connection State (stores full Airbyte state blob per connection)
CREATE TABLE pipeline.connection_state (
    connection_id UUID PRIMARY KEY REFERENCES pipeline.connections(id) ON DELETE CASCADE,
    state_type pipeline.state_type NOT NULL DEFAULT 'stream',
    state_blob JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job Logs (incremental log capture during sync execution)
CREATE TABLE pipeline.job_logs (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES pipeline.jobs(id) ON DELETE CASCADE,
    log_line TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_job_logs_job_id ON pipeline.job_logs(job_id);

-- Stats Rollups (pre-computed aggregations for dashboard charts)
CREATE TABLE pipeline.stats_rollups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    connection_id UUID NOT NULL,
    bucket_start TIMESTAMPTZ NOT NULL,
    bucket_size pipeline.bucket_size NOT NULL,
    syncs_total INT NOT NULL DEFAULT 0,
    syncs_succeeded INT NOT NULL DEFAULT 0,
    syncs_failed INT NOT NULL DEFAULT 0,
    records_read BIGINT NOT NULL DEFAULT 0,
    bytes_synced BIGINT NOT NULL DEFAULT 0,
    total_duration_ms BIGINT NOT NULL DEFAULT 0,
    avg_duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(connection_id, bucket_start, bucket_size)
);
CREATE INDEX idx_stats_rollups_workspace_bucket
    ON pipeline.stats_rollups(workspace_id, bucket_start, bucket_size);
CREATE INDEX idx_stats_rollups_connection_bucket
    ON pipeline.stats_rollups(connection_id, bucket_start, bucket_size);

-- Connector Tasks
CREATE TABLE pipeline.connector_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    task_type pipeline.connector_task_type NOT NULL,
    status pipeline.connector_task_status NOT NULL DEFAULT 'pending',
    payload JSONB NOT NULL,
    result JSONB,
    error_message TEXT,
    worker_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX idx_connector_tasks_status ON pipeline.connector_tasks(status);
CREATE INDEX idx_connector_tasks_workspace_id ON pipeline.connector_tasks(workspace_id);

-- Workspace Settings (pipeline-owned, no FK to workspace module)
CREATE TABLE pipeline.workspace_settings (
    workspace_id UUID PRIMARY KEY,
    max_jobs_per_workspace INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Stream Generations (per-stream generation_id counter for Airbyte protocol)
CREATE TABLE pipeline.stream_generations (
    connection_id UUID NOT NULL REFERENCES pipeline.connections(id) ON DELETE CASCADE,
    stream_namespace TEXT NOT NULL DEFAULT '',
    stream_name TEXT NOT NULL,
    generation_id BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (connection_id, stream_namespace, stream_name)
);
CREATE INDEX idx_stream_generations_connection_id ON pipeline.stream_generations(connection_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pipeline.stream_generations;
DROP TABLE IF EXISTS pipeline.workspace_settings;
DROP TABLE IF EXISTS pipeline.connector_tasks;
DROP TABLE IF EXISTS pipeline.stats_rollups;
DROP TABLE IF EXISTS pipeline.job_logs;
DROP TABLE IF EXISTS pipeline.connection_state;
DROP TABLE IF EXISTS pipeline.configured_catalogs;
DROP TABLE IF EXISTS pipeline.catalog_discoveries;
DROP TABLE IF EXISTS pipeline.job_attempts;
DROP TABLE IF EXISTS pipeline.jobs;
DROP TABLE IF EXISTS pipeline.connections;
DROP TABLE IF EXISTS pipeline.destinations;
DROP TABLE IF EXISTS pipeline.sources;
DROP TABLE IF EXISTS pipeline.managed_connectors;
DROP TABLE IF EXISTS pipeline.repository_connectors;
DROP TABLE IF EXISTS pipeline.repositories;
DROP TYPE IF EXISTS pipeline.release_stage;
DROP TYPE IF EXISTS pipeline.source_type;
DROP TYPE IF EXISTS pipeline.support_level;
DROP TYPE IF EXISTS pipeline.repository_status;
DROP TYPE IF EXISTS pipeline.connector_task_status;
DROP TYPE IF EXISTS pipeline.connector_task_type;
DROP TYPE IF EXISTS pipeline.connector_type;
DROP TYPE IF EXISTS pipeline.bucket_size;
DROP TYPE IF EXISTS pipeline.state_type;
DROP TYPE IF EXISTS pipeline.namespace_definition;
DROP TYPE IF EXISTS pipeline.job_type;
DROP TYPE IF EXISTS pipeline.job_status;
DROP TYPE IF EXISTS pipeline.schema_change_policy;
DROP TYPE IF EXISTS pipeline.connection_status;
DROP SCHEMA IF EXISTS pipeline;
-- +goose StatementEnd
