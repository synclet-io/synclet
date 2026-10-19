-- +goose Up
-- +goose StatementBegin
ALTER TABLE workspace.workspaces DROP COLUMN IF EXISTS max_jobs_per_workspace;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
