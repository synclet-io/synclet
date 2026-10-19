-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS pipeline.schema_changes;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
