-- +goose Up
-- +goose StatementBegin
CREATE TYPE pipeline.job_log_segment_status AS ENUM ('open', 'archived');

ALTER TABLE pipeline.job_logs ADD COLUMN attempt INT NOT NULL DEFAULT 1;

-- One row per job attempt that produced log lines: line-cap counters while the
-- attempt runs, and the archive object once its lines move out of job_logs.
CREATE TABLE pipeline.job_log_segments (
    job_id UUID NOT NULL REFERENCES pipeline.jobs(id) ON DELETE CASCADE,
    attempt INT NOT NULL,
    status pipeline.job_log_segment_status NOT NULL DEFAULT 'open',
    line_count INT NOT NULL DEFAULT 0,
    dropped_count INT NOT NULL DEFAULT 0,
    marker_log_id BIGINT,
    first_log_id BIGINT,
    last_log_id BIGINT,
    archive_key TEXT,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ,
    PRIMARY KEY (job_id, attempt)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE pipeline.job_log_segments;
ALTER TABLE pipeline.job_logs DROP COLUMN attempt;
DROP TYPE pipeline.job_log_segment_status;
-- +goose StatementEnd
