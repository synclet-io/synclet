-- +goose Up
-- +goose StatementBegin
CREATE TYPE auth.oauth_access_level AS ENUM ('read', 'actions', 'write');

CREATE TABLE auth.oauth_clients (
    id              UUID PRIMARY KEY,
    client_id       TEXT        NOT NULL UNIQUE,
    name            TEXT        NOT NULL,
    redirect_uris   TEXT[]      NOT NULL,
    last_granted_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_clients_created_at ON auth.oauth_clients (created_at);

CREATE TABLE auth.oauth_codes (
    id             UUID PRIMARY KEY,
    code_hash      TEXT                    NOT NULL UNIQUE,
    client_id      TEXT                    NOT NULL REFERENCES auth.oauth_clients (client_id) ON DELETE CASCADE,
    redirect_uri   TEXT                    NOT NULL,
    code_challenge TEXT                    NOT NULL,
    user_id        UUID                    NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    workspace_id   UUID                    NOT NULL,
    level          auth.oauth_access_level NOT NULL,
    expires_at     TIMESTAMPTZ             NOT NULL,
    created_at     TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_codes_expires_at ON auth.oauth_codes (expires_at);

CREATE TABLE auth.oauth_grants (
    id           UUID PRIMARY KEY,
    client_id    TEXT                    NOT NULL REFERENCES auth.oauth_clients (client_id) ON DELETE CASCADE,
    user_id      UUID                    NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    workspace_id UUID                    NOT NULL,
    level        auth.oauth_access_level NOT NULL,
    created_at   TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_grants_user_id ON auth.oauth_grants (user_id);
CREATE INDEX idx_oauth_grants_workspace_id ON auth.oauth_grants (workspace_id);

CREATE TABLE auth.oauth_refresh_tokens (
    id         UUID PRIMARY KEY,
    grant_id   UUID        NOT NULL REFERENCES auth.oauth_grants (id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    used_at    TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_refresh_tokens_grant_id ON auth.oauth_refresh_tokens (grant_id);
CREATE INDEX idx_oauth_refresh_tokens_expires_at ON auth.oauth_refresh_tokens (expires_at);

ALTER TABLE auth.api_keys
    ADD COLUMN oauth_grant_id UUID REFERENCES auth.oauth_grants (id) ON DELETE CASCADE,
    ADD COLUMN oauth_level    auth.oauth_access_level;

CREATE INDEX idx_api_keys_oauth_grant_id ON auth.api_keys (oauth_grant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE auth.api_keys DROP COLUMN oauth_level, DROP COLUMN oauth_grant_id;
DROP TABLE auth.oauth_refresh_tokens;
DROP TABLE auth.oauth_grants;
DROP TABLE auth.oauth_codes;
DROP TABLE auth.oauth_clients;
DROP TYPE auth.oauth_access_level;
-- +goose StatementEnd
