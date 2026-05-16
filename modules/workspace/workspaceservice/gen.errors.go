package workspaceservice

import (
	fmt "fmt"
	// user code 'imports'
	// end user code 'imports'
)

type NotFoundError string

func (n NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", string(n))
}

type AlreadyExistsError string

func (a AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists", string(a))
}

const (
	ErrWorkspaceNotFound      = NotFoundError("Workspace")
	ErrWorkspaceAlreadyExists = AlreadyExistsError("Workspace")
)
const (
	ErrWorkspaceMemberNotFound      = NotFoundError("WorkspaceMember")
	ErrWorkspaceMemberAlreadyExists = AlreadyExistsError("WorkspaceMember")
)
const (
	ErrWorkspaceInviteNotFound      = NotFoundError("WorkspaceInvite")
	ErrWorkspaceInviteAlreadyExists = AlreadyExistsError("WorkspaceInvite")
)
const (
	ErrWorkspaceEventNotFound      = NotFoundError("WorkspaceEvent")
	ErrWorkspaceEventAlreadyExists = AlreadyExistsError("WorkspaceEvent")
)
const (
	ErrWorkspaceCreatedEventDataNotFound      = NotFoundError("WorkspaceCreatedEventData")
	ErrWorkspaceCreatedEventDataAlreadyExists = AlreadyExistsError("WorkspaceCreatedEventData")
)
const (
	ErrWorkspaceUpdatedEventDataNotFound      = NotFoundError("WorkspaceUpdatedEventData")
	ErrWorkspaceUpdatedEventDataAlreadyExists = AlreadyExistsError("WorkspaceUpdatedEventData")
)
const (
	ErrWorkspaceDeletedEventDataNotFound      = NotFoundError("WorkspaceDeletedEventData")
	ErrWorkspaceDeletedEventDataAlreadyExists = AlreadyExistsError("WorkspaceDeletedEventData")
)
