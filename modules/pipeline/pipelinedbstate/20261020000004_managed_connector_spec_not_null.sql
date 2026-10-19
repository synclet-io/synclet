-- +goose Up
-- +goose StatementBegin
UPDATE pipeline.managed_connectors SET spec = '{}' WHERE spec IS NULL;
ALTER TABLE pipeline.managed_connectors
    ALTER COLUMN spec SET DEFAULT '{}',
    ALTER COLUMN spec SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pipeline.managed_connectors
    ALTER COLUMN spec DROP NOT NULL,
    ALTER COLUMN spec DROP DEFAULT;
-- +goose StatementEnd
