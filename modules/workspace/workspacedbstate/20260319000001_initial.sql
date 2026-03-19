-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS workspace;

CREATE TABLE workspace.workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TYPE workspace.member_role AS ENUM ('admin', 'editor', 'viewer');

CREATE TABLE workspace.workspace_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace.workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role workspace.member_role NOT NULL DEFAULT 'viewer',
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, user_id)
);
CREATE INDEX idx_workspace_members_user_id ON workspace.workspace_members(user_id);

CREATE TYPE workspace.invite_status AS ENUM ('pending', 'accepted', 'declined', 'revoked');

CREATE TABLE workspace.workspace_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspace.workspaces(id) ON DELETE CASCADE,
    inviter_user_id UUID NOT NULL,
    email TEXT NOT NULL,
    role workspace.member_role NOT NULL DEFAULT 'viewer',
    token TEXT NOT NULL UNIQUE,
    status workspace.invite_status NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspace_invites_workspace_id ON workspace.workspace_invites(workspace_id);
CREATE INDEX idx_workspace_invites_email ON workspace.workspace_invites(email);
CREATE INDEX idx_workspace_invites_token ON workspace.workspace_invites(token);
CREATE UNIQUE INDEX idx_workspace_invites_unique_pending
    ON workspace.workspace_invites(workspace_id, email)
    WHERE status = 'pending';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS workspace.workspace_invites;
DROP TABLE IF EXISTS workspace.workspace_members;
DROP TABLE IF EXISTS workspace.workspaces;
DROP TYPE IF EXISTS workspace.invite_status;
DROP TYPE IF EXISTS workspace.member_role;
DROP SCHEMA IF EXISTS workspace;
-- +goose StatementEnd
