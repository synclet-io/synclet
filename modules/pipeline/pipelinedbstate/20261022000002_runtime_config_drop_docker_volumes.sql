-- +goose Up
-- +goose StatementBegin
UPDATE pipeline.sources
SET runtime_config = runtime_config - 'docker_volumes'
WHERE jsonb_typeof(runtime_config) = 'object'
  AND runtime_config ? 'docker_volumes';

UPDATE pipeline.destinations
SET runtime_config = runtime_config - 'docker_volumes'
WHERE jsonb_typeof(runtime_config) = 'object'
  AND runtime_config ? 'docker_volumes';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
