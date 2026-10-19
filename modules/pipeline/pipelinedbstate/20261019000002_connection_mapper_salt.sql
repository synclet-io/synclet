-- +goose Up
-- +goose StatementBegin
ALTER TABLE pipeline.connections ADD COLUMN IF NOT EXISTS mapper_salt TEXT NOT NULL DEFAULT replace(gen_random_uuid()::text, '-', '');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pipeline.connections DROP COLUMN IF EXISTS mapper_salt;
-- +goose StatementEnd
