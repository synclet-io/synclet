-- +goose Up
-- +goose StatementBegin
UPDATE pipeline.repositories SET status = 'synced' WHERE status = 'syncing';
ALTER TABLE pipeline.repositories ALTER COLUMN status DROP DEFAULT;

ALTER TYPE pipeline.repository_status RENAME TO repository_status_old;
CREATE TYPE pipeline.repository_status AS ENUM ('synced', 'failed');
ALTER TABLE pipeline.repositories ALTER COLUMN status TYPE pipeline.repository_status USING status::text::pipeline.repository_status;
ALTER TABLE pipeline.repositories ALTER COLUMN status SET DEFAULT 'synced';
DROP TYPE pipeline.repository_status_old;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TYPE pipeline.repository_status RENAME TO repository_status_old;
CREATE TYPE pipeline.repository_status AS ENUM ('syncing', 'synced', 'failed');
ALTER TABLE pipeline.repositories ALTER COLUMN status TYPE pipeline.repository_status USING status::text::pipeline.repository_status;
ALTER TABLE pipeline.repositories ALTER COLUMN status SET DEFAULT 'syncing';
DROP TYPE pipeline.repository_status_old;
-- +goose StatementEnd
