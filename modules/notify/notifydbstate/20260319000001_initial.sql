-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS notify;

CREATE TABLE notify.webhooks (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    url TEXT NOT NULL,
    events JSONB NOT NULL DEFAULT '[]',
    secret TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_workspace_id ON notify.webhooks(workspace_id);
CREATE INDEX idx_webhooks_enabled ON notify.webhooks(enabled);

CREATE TYPE notify.channel_type AS ENUM ('slack', 'email', 'telegram');
CREATE TYPE notify.notification_condition AS ENUM ('on_failure', 'on_consecutive_failures', 'on_zero_records');

CREATE TABLE notify.notification_channels (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    name TEXT NOT NULL,
    channel_type notify.channel_type NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notification_channels_workspace_id ON notify.notification_channels(workspace_id);
CREATE INDEX idx_notification_channels_channel_type ON notify.notification_channels(channel_type);
CREATE INDEX idx_notification_channels_enabled ON notify.notification_channels(enabled);

CREATE TABLE notify.notification_rules (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    channel_id UUID NOT NULL REFERENCES notify.notification_channels(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    condition notify.notification_condition NOT NULL,
    condition_value INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notification_rules_workspace_id ON notify.notification_rules(workspace_id);
CREATE INDEX idx_notification_rules_channel_id ON notify.notification_rules(channel_id);
CREATE INDEX idx_notification_rules_connection_id ON notify.notification_rules(connection_id);
CREATE INDEX idx_notification_rules_enabled ON notify.notification_rules(enabled);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notify.notification_rules;
DROP TABLE IF EXISTS notify.notification_channels;
DROP TYPE IF EXISTS notify.notification_condition;
DROP TYPE IF EXISTS notify.channel_type;
DROP TABLE IF EXISTS notify.webhooks;
DROP SCHEMA IF EXISTS notify;
-- +goose StatementEnd
