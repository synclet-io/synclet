-- +goose Up
-- +goose StatementBegin
CREATE SEQUENCE IF NOT EXISTS pipeline.job_sync_id_seq AS BIGINT;

SELECT setval('pipeline.job_sync_id_seq', (SELECT COUNT(*) + 1 FROM pipeline.jobs), false);

ALTER TABLE pipeline.jobs ADD COLUMN IF NOT EXISTS sync_id BIGINT UNIQUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE pipeline.jobs DROP COLUMN IF EXISTS sync_id;
DROP SEQUENCE IF EXISTS pipeline.job_sync_id_seq;
-- +goose StatementEnd
