-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS secret;
CREATE TABLE secret.secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    encrypted_value BYTEA NOT NULL,
    salt BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    owner_type TEXT NOT NULL,
    owner_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_secrets_owner ON secret.secrets (owner_type, owner_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS secret.secrets;
DROP SCHEMA IF EXISTS secret;
-- +goose StatementEnd
