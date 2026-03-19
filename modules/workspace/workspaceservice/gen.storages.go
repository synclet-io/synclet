package workspaceservice

import (
	context "context"

	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	// user code 'imports'
	// end user code 'imports'
)

type Storage interface {
	Workspaces() WorkspacesStorage
	WorkspaceMembers() WorkspaceMembersStorage
	WorkspaceInvites() WorkspaceInvitesStorage
	IdempotencyKeys() idempotency.Storage
	ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error
	WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error
}
type WorkspacesStorage dbutil.EntityStorage[Workspace, WorkspaceFilter]
type WorkspaceMembersStorage dbutil.EntityStorage[WorkspaceMember, WorkspaceMemberFilter]
type WorkspaceInvitesStorage dbutil.EntityStorage[WorkspaceInvite, WorkspaceInviteFilter]
