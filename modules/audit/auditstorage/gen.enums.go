package auditstorage

import (
	fmt "fmt"

	auditservice "github.com/synclet-io/synclet/modules/audit/auditservice"
	// user code 'imports'
	// end user code 'imports'
)

const (
	actionCreate  = "create"
	actionUpdate  = "update"
	actionDelete  = "delete"
	actionEnable  = "enable"
	actionDisable = "disable"
	actionTest    = "test"
	actionSync    = "sync"
	actionLogin   = "login"
	actionLogout  = "logout"
)

func convertActionToDB(actionValue auditservice.Action) (string, error) {
	result, ok := map[auditservice.Action]string{
		auditservice.ActionCreate:  actionCreate,
		auditservice.ActionUpdate:  actionUpdate,
		auditservice.ActionDelete:  actionDelete,
		auditservice.ActionEnable:  actionEnable,
		auditservice.ActionDisable: actionDisable,
		auditservice.ActionTest:    actionTest,
		auditservice.ActionSync:    actionSync,
		auditservice.ActionLogin:   actionLogin,
		auditservice.ActionLogout:  actionLogout,
	}[actionValue]
	if !ok {
		return "", fmt.Errorf("unknown Action value: %d", actionValue)
	}
	return result, nil
}

func convertActionFromDB(actionValue string) (auditservice.Action, error) {
	result, ok := map[string]auditservice.Action{
		actionCreate:  auditservice.ActionCreate,
		actionUpdate:  auditservice.ActionUpdate,
		actionDelete:  auditservice.ActionDelete,
		actionEnable:  auditservice.ActionEnable,
		actionDisable: auditservice.ActionDisable,
		actionTest:    auditservice.ActionTest,
		actionSync:    auditservice.ActionSync,
		actionLogin:   auditservice.ActionLogin,
		actionLogout:  auditservice.ActionLogout,
	}[actionValue]
	if !ok {
		return 0, fmt.Errorf("unknown Action db value: %s", actionValue)
	}
	return result, nil
}

const (
	resourceTypeSource              = "source"
	resourceTypeDestination         = "destination"
	resourceTypeConnection          = "connection"
	resourceTypeConnector           = "connector"
	resourceTypeRepository          = "repository"
	resourceTypeNotificationChannel = "notification_channel"
	resourceTypeNotificationRule    = "notification_rule"
	resourceTypeWebhook             = "webhook"
	resourceTypeWorkspace           = "workspace"
	resourceTypeWorkspaceMember     = "workspace_member"
	resourceTypeWorkspaceInvite     = "workspace_invite"
	resourceTypeAPIKey              = "api_key"
	resourceTypeUser                = "user"
)

func convertResourceTypeToDB(resourceTypeValue auditservice.ResourceType) (string, error) {
	result, ok := map[auditservice.ResourceType]string{
		auditservice.ResourceTypeSource:              resourceTypeSource,
		auditservice.ResourceTypeDestination:         resourceTypeDestination,
		auditservice.ResourceTypeConnection:          resourceTypeConnection,
		auditservice.ResourceTypeConnector:           resourceTypeConnector,
		auditservice.ResourceTypeRepository:          resourceTypeRepository,
		auditservice.ResourceTypeNotificationChannel: resourceTypeNotificationChannel,
		auditservice.ResourceTypeNotificationRule:    resourceTypeNotificationRule,
		auditservice.ResourceTypeWebhook:             resourceTypeWebhook,
		auditservice.ResourceTypeWorkspace:           resourceTypeWorkspace,
		auditservice.ResourceTypeWorkspaceMember:     resourceTypeWorkspaceMember,
		auditservice.ResourceTypeWorkspaceInvite:     resourceTypeWorkspaceInvite,
		auditservice.ResourceTypeAPIKey:              resourceTypeAPIKey,
		auditservice.ResourceTypeUser:                resourceTypeUser,
	}[resourceTypeValue]
	if !ok {
		return "", fmt.Errorf("unknown ResourceType value: %d", resourceTypeValue)
	}
	return result, nil
}

func convertResourceTypeFromDB(resourceTypeValue string) (auditservice.ResourceType, error) {
	result, ok := map[string]auditservice.ResourceType{
		resourceTypeSource:              auditservice.ResourceTypeSource,
		resourceTypeDestination:         auditservice.ResourceTypeDestination,
		resourceTypeConnection:          auditservice.ResourceTypeConnection,
		resourceTypeConnector:           auditservice.ResourceTypeConnector,
		resourceTypeRepository:          auditservice.ResourceTypeRepository,
		resourceTypeNotificationChannel: auditservice.ResourceTypeNotificationChannel,
		resourceTypeNotificationRule:    auditservice.ResourceTypeNotificationRule,
		resourceTypeWebhook:             auditservice.ResourceTypeWebhook,
		resourceTypeWorkspace:           auditservice.ResourceTypeWorkspace,
		resourceTypeWorkspaceMember:     auditservice.ResourceTypeWorkspaceMember,
		resourceTypeWorkspaceInvite:     auditservice.ResourceTypeWorkspaceInvite,
		resourceTypeAPIKey:              auditservice.ResourceTypeAPIKey,
		resourceTypeUser:                auditservice.ResourceTypeUser,
	}[resourceTypeValue]
	if !ok {
		return 0, fmt.Errorf("unknown ResourceType db value: %s", resourceTypeValue)
	}
	return result, nil
}

const (
	actorTypeUser   = "user"
	actorTypeAPIKey = "api_key"
	actorTypeSystem = "system"
)

func convertActorTypeToDB(actorTypeValue auditservice.ActorType) (string, error) {
	result, ok := map[auditservice.ActorType]string{
		auditservice.ActorTypeUser:   actorTypeUser,
		auditservice.ActorTypeAPIKey: actorTypeAPIKey,
		auditservice.ActorTypeSystem: actorTypeSystem,
	}[actorTypeValue]
	if !ok {
		return "", fmt.Errorf("unknown ActorType value: %d", actorTypeValue)
	}
	return result, nil
}

func convertActorTypeFromDB(actorTypeValue string) (auditservice.ActorType, error) {
	result, ok := map[string]auditservice.ActorType{
		actorTypeUser:   auditservice.ActorTypeUser,
		actorTypeAPIKey: auditservice.ActorTypeAPIKey,
		actorTypeSystem: auditservice.ActorTypeSystem,
	}[actorTypeValue]
	if !ok {
		return 0, fmt.Errorf("unknown ActorType db value: %s", actorTypeValue)
	}
	return result, nil
}
