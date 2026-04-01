-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

ALTER TABLE auth.users ALTER COLUMN email TYPE CITEXT;
ALTER TABLE auth.oidc_identities ALTER COLUMN email TYPE CITEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE auth.oidc_identities ALTER COLUMN email TYPE TEXT;
ALTER TABLE auth.users ALTER COLUMN email TYPE TEXT;
-- +goose StatementEnd
