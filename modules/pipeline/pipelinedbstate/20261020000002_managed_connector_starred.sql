-- +goose Up
-- +goose StatementBegin
ALTER TABLE pipeline.managed_connectors ADD COLUMN IF NOT EXISTS starred BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pipeline.managed_connectors DROP COLUMN IF EXISTS starred;
-- +goose StatementEnd
