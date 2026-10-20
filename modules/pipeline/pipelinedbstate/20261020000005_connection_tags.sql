-- +goose Up
-- +goose StatementBegin
ALTER TABLE pipeline.connections ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS connections_tags_gin_idx ON pipeline.connections USING GIN (tags);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS pipeline.connections_tags_gin_idx;
ALTER TABLE pipeline.connections DROP COLUMN IF EXISTS tags;
-- +goose StatementEnd
