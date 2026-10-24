-- +goose Up
-- +goose StatementBegin
CREATE TABLE auth.rate_limits (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key        TEXT        NOT NULL UNIQUE,
    count      INTEGER     NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_rate_limits_expires_at ON auth.rate_limits (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE auth.rate_limits;
-- +goose StatementEnd
