-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS audit;

CREATE TYPE audit.action AS ENUM (
    'create',
    'update',
    'delete',
    'enable',
    'disable',
    'test',
    'sync',
    'login',
    'logout'
);

CREATE TYPE audit.resource_type AS ENUM (
    'source',
    'destination',
    'connection',
    'connector',
    'repository',
    'notification_channel',
    'notification_rule',
    'webhook',
    'workspace',
    'workspace_member',
    'workspace_invite',
    'api_key',
    'user'
);

CREATE TYPE audit.actor_type AS ENUM (
    'user',
    'api_key',
    'system'
);

CREATE TABLE audit.audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    actor_type audit.actor_type NOT NULL,
    actor_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    actor_label TEXT NOT NULL,
    action audit.action NOT NULL,
    resource_type audit.resource_type NOT NULL,
    resource_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    resource_label TEXT NOT NULL,
    diff_json TEXT NOT NULL DEFAULT '[]',
    diff_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_workspace_created
    ON audit.audit_events (workspace_id, created_at DESC);

CREATE INDEX idx_audit_events_resource
    ON audit.audit_events (workspace_id, resource_type, resource_id);

CREATE INDEX idx_audit_events_actor
    ON audit.audit_events (workspace_id, actor_type, actor_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit.audit_events;
DROP TYPE IF EXISTS audit.actor_type;
DROP TYPE IF EXISTS audit.resource_type;
DROP TYPE IF EXISTS audit.action;
DROP SCHEMA IF EXISTS audit;
-- +goose StatementEnd
