-- +goose Up
-- +goose StatementBegin
ALTER TABLE pipeline.connections ADD COLUMN IF NOT EXISTS status_reason TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pipeline.connections DROP COLUMN IF EXISTS status_reason;
-- +goose StatementEnd
