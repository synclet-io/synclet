-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit.retention_settings (
    workspace_id UUID PRIMARY KEY,
    retention_days INTEGER NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit.retention_settings;
-- +goose StatementEnd
