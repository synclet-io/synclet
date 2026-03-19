package workspacestorage

import (
	context "context"
	strconv "strconv"

	xxhash "github.com/cespare/xxhash"
	logging "github.com/go-pnp/go-pnp/logging"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	txoutbox "github.com/saturn4er/boilerplate-go/lib/txoutbox"
	gorm "gorm.io/gorm"
	clause "gorm.io/gorm/clause"

	workspacesvc "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	// user code 'imports'
	// end user code 'imports'
)

type Storages struct {
	db         *gorm.DB
	logger     *logging.Logger
	processors []txoutbox.MessageProcessor
}

var _ workspacesvc.Storage = &Storages{}

func (s Storages) Workspaces() workspacesvc.WorkspacesStorage {
	return NewWorkspacesStorage(s.db, s.logger)
}
func (s Storages) WorkspaceMembers() workspacesvc.WorkspaceMembersStorage {
	return NewWorkspaceMembersStorage(s.db, s.logger)
}
func (s Storages) WorkspaceInvites() workspacesvc.WorkspaceInvitesStorage {
	return NewWorkspaceInvitesStorage(s.db, s.logger)
}

func (s Storages) IdempotencyKeys() idempotency.Storage {
	return idempotency.GormStorage{
		DB: s.db,
	}

}

func (s *Storages) WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error {
	hasher := xxhash.New()
	hasher.Write([]byte(scope))
	hasher.Write([]byte{':'})
	hasher.Write(strconv.AppendInt(nil, lockID, 10))

	result := s.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", int64(hasher.Sum64()))
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (s Storages) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx workspacesvc.Storage) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return cb(ctx, &Storages{db: tx, logger: s.logger, processors: s.processors})
	})
}

func NewStorages(db *gorm.DB, logger *logging.Logger, processors []txoutbox.MessageProcessor) *Storages {
	return &Storages{db: db, logger: logger, processors: processors}
}

func NewWorkspacesStorage(db *gorm.DB, logger *logging.Logger) workspacesvc.WorkspacesStorage {
	return dbutil.GormEntityStorage[workspacesvc.Workspace, dbWorkspace, workspacesvc.WorkspaceFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapWorkspaceQueryError,
		ConvertToInternal: convertWorkspaceToDB,
		ConvertToExternal: convertWorkspaceFromDB,
		BuildFilterExpression: func(filter *workspacesvc.WorkspaceFilter) (clause.Expression, error) {
			return buildWorkspaceFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			workspacesvc.WorkspaceFieldID:        {Name: "id"},
			workspacesvc.WorkspaceFieldName:      {Name: "name"},
			workspacesvc.WorkspaceFieldSlug:      {Name: "slug"},
			workspacesvc.WorkspaceFieldCreatedAt: {Name: "created_at"},
			workspacesvc.WorkspaceFieldUpdatedAt: {Name: "updated_at"},
		},
		LockScope: "workspace.Workspaces",
	}
}

func NewWorkspaceMembersStorage(db *gorm.DB, logger *logging.Logger) workspacesvc.WorkspaceMembersStorage {
	return dbutil.GormEntityStorage[workspacesvc.WorkspaceMember, dbWorkspaceMember, workspacesvc.WorkspaceMemberFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapWorkspaceMemberQueryError,
		ConvertToInternal: convertWorkspaceMemberToDB,
		ConvertToExternal: convertWorkspaceMemberFromDB,
		BuildFilterExpression: func(filter *workspacesvc.WorkspaceMemberFilter) (clause.Expression, error) {
			return buildWorkspaceMemberFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			workspacesvc.WorkspaceMemberFieldID:          {Name: "id"},
			workspacesvc.WorkspaceMemberFieldWorkspaceID: {Name: "workspace_id"},
			workspacesvc.WorkspaceMemberFieldUserID:      {Name: "user_id"},
			workspacesvc.WorkspaceMemberFieldRole:        {Name: "role"},
			workspacesvc.WorkspaceMemberFieldJoinedAt:    {Name: "joined_at"},
		},
		LockScope: "workspace.WorkspaceMembers",
	}
}

func NewWorkspaceInvitesStorage(db *gorm.DB, logger *logging.Logger) workspacesvc.WorkspaceInvitesStorage {
	return dbutil.GormEntityStorage[workspacesvc.WorkspaceInvite, dbWorkspaceInvite, workspacesvc.WorkspaceInviteFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapWorkspaceInviteQueryError,
		ConvertToInternal: convertWorkspaceInviteToDB,
		ConvertToExternal: convertWorkspaceInviteFromDB,
		BuildFilterExpression: func(filter *workspacesvc.WorkspaceInviteFilter) (clause.Expression, error) {
			return buildWorkspaceInviteFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			workspacesvc.WorkspaceInviteFieldID:            {Name: "id"},
			workspacesvc.WorkspaceInviteFieldWorkspaceID:   {Name: "workspace_id"},
			workspacesvc.WorkspaceInviteFieldInviterUserID: {Name: "inviter_user_id"},
			workspacesvc.WorkspaceInviteFieldEmail:         {Name: "email"},
			workspacesvc.WorkspaceInviteFieldRole:          {Name: "role"},
			workspacesvc.WorkspaceInviteFieldToken:         {Name: "token"},
			workspacesvc.WorkspaceInviteFieldStatus:        {Name: "status"},
			workspacesvc.WorkspaceInviteFieldExpiresAt:     {Name: "expires_at"},
			workspacesvc.WorkspaceInviteFieldCreatedAt:     {Name: "created_at"},
			workspacesvc.WorkspaceInviteFieldUpdatedAt:     {Name: "updated_at"},
		},
		LockScope: "workspace.WorkspaceInvites",
	}
}
