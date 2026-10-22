-- +goose Up
-- +goose StatementBegin
UPDATE pipeline.sources
SET runtime_config = runtime_config - 'service_account_name'
WHERE jsonb_typeof(runtime_config) = 'object'
  AND runtime_config ? 'service_account_name';

UPDATE pipeline.destinations
SET runtime_config = runtime_config - 'service_account_name'
WHERE jsonb_typeof(runtime_config) = 'object'
  AND runtime_config ? 'service_account_name';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
