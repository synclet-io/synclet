package auditservice

import (
	strconv "strconv"
	// user code 'imports'
	// end user code 'imports'
)

type Action byte

const (
	ActionCreate Action = iota + 1
	ActionUpdate
	ActionDelete
	ActionEnable
	ActionDisable
	ActionTest
	ActionSync
	ActionLogin
	ActionLogout
)

// user code 'Action methods'
// end user code 'Action methods'
func (a Action) IsValid() bool {
	return a > 0 && a < 10
}
func (a Action) IsCreate() bool {
	return a == ActionCreate
}
func (a Action) IsUpdate() bool {
	return a == ActionUpdate
}
func (a Action) IsDelete() bool {
	return a == ActionDelete
}
func (a Action) IsEnable() bool {
	return a == ActionEnable
}
func (a Action) IsDisable() bool {
	return a == ActionDisable
}
func (a Action) IsTest() bool {
	return a == ActionTest
}
func (a Action) IsSync() bool {
	return a == ActionSync
}
func (a Action) IsLogin() bool {
	return a == ActionLogin
}
func (a Action) IsLogout() bool {
	return a == ActionLogout
}
func (a Action) String() string {
	const names = "CreateUpdateDeleteEnableDisableTestSyncLoginLogout"

	var indexes = [...]int32{0, 6, 12, 18, 24, 31, 35, 39, 44, 50}
	if a < 1 || a > 9 {
		return "Action(" + strconv.FormatInt(int64(a), 10) + ")"
	}

	return names[indexes[a-1]:indexes[a]]
}

type ResourceType byte

const (
	ResourceTypeSource ResourceType = iota + 1
	ResourceTypeDestination
	ResourceTypeConnection
	ResourceTypeConnector
	ResourceTypeRepository
	ResourceTypeNotificationChannel
	ResourceTypeNotificationRule
	ResourceTypeWebhook
	ResourceTypeWorkspace
	ResourceTypeWorkspaceMember
	ResourceTypeWorkspaceInvite
	ResourceTypeAPIKey
	ResourceTypeUser
)

// user code 'ResourceType methods'
// end user code 'ResourceType methods'
func (r ResourceType) IsValid() bool {
	return r > 0 && r < 14
}
func (r ResourceType) IsSource() bool {
	return r == ResourceTypeSource
}
func (r ResourceType) IsDestination() bool {
	return r == ResourceTypeDestination
}
func (r ResourceType) IsConnection() bool {
	return r == ResourceTypeConnection
}
func (r ResourceType) IsConnector() bool {
	return r == ResourceTypeConnector
}
func (r ResourceType) IsRepository() bool {
	return r == ResourceTypeRepository
}
func (r ResourceType) IsNotificationChannel() bool {
	return r == ResourceTypeNotificationChannel
}
func (r ResourceType) IsNotificationRule() bool {
	return r == ResourceTypeNotificationRule
}
func (r ResourceType) IsWebhook() bool {
	return r == ResourceTypeWebhook
}
func (r ResourceType) IsWorkspace() bool {
	return r == ResourceTypeWorkspace
}
func (r ResourceType) IsWorkspaceMember() bool {
	return r == ResourceTypeWorkspaceMember
}
func (r ResourceType) IsWorkspaceInvite() bool {
	return r == ResourceTypeWorkspaceInvite
}
func (r ResourceType) IsAPIKey() bool {
	return r == ResourceTypeAPIKey
}
func (r ResourceType) IsUser() bool {
	return r == ResourceTypeUser
}
func (r ResourceType) String() string {
	const names = "SourceDestinationConnectionConnectorRepositoryNotificationChannelNotificationRuleWebhookWorkspaceWorkspaceMemberWorkspaceInviteAPIKeyUser"

	var indexes = [...]int32{0, 6, 17, 27, 36, 46, 65, 81, 88, 97, 112, 127, 133, 137}
	if r < 1 || r > 13 {
		return "ResourceType(" + strconv.FormatInt(int64(r), 10) + ")"
	}

	return names[indexes[r-1]:indexes[r]]
}

type ActorType byte

const (
	ActorTypeUser ActorType = iota + 1
	ActorTypeAPIKey
	ActorTypeSystem
)

// user code 'ActorType methods'
// end user code 'ActorType methods'
func (a ActorType) IsValid() bool {
	return a > 0 && a < 4
}
func (a ActorType) IsUser() bool {
	return a == ActorTypeUser
}
func (a ActorType) IsAPIKey() bool {
	return a == ActorTypeAPIKey
}
func (a ActorType) IsSystem() bool {
	return a == ActorTypeSystem
}
func (a ActorType) String() string {
	const names = "UserAPIKeySystem"

	var indexes = [...]int32{0, 4, 10, 16}
	if a < 1 || a > 3 {
		return "ActorType(" + strconv.FormatInt(int64(a), 10) + ")"
	}

	return names[indexes[a-1]:indexes[a]]
}
