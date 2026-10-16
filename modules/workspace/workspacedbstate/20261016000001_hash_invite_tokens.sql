-- +goose Up
-- +goose StatementBegin
UPDATE workspace.workspace_invites SET token = encode(sha256(convert_to(token, 'UTF8')), 'hex');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE workspace.workspace_invites SET status = 'revoked' WHERE status = 'pending';
-- +goose StatementEnd
