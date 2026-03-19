import { createClient } from '@connectrpc/connect'
import { AuthService } from '@/gen/synclet/publicapi/auth/v1/auth_pb'

import { ConfiguredStreamSchema, ConnectionService, ConnectorTaskService, DestinationService, JobService, SourceService } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { NotificationService } from '@/gen/synclet/publicapi/notify/v1/notify_pb'
import { ConnectorRegistryService } from '@/gen/synclet/publicapi/registry/v1/registry_pb'
import { WorkspaceService } from '@/gen/synclet/publicapi/workspace/v1/workspace_pb'
import { transport } from './transport'

export const authClient = createClient(AuthService, transport)
export const workspaceClient = createClient(WorkspaceService, transport)
export const sourceClient = createClient(SourceService, transport)
export const destinationClient = createClient(DestinationService, transport)
export const connectionClient = createClient(ConnectionService, transport)
export const jobClient = createClient(JobService, transport)
export const connectorTaskClient = createClient(ConnectorTaskService, transport)
export const notificationClient = createClient(NotificationService, transport)
export const registryClient = createClient(ConnectorRegistryService, transport)

export { create } from '@bufbuild/protobuf'
export { ConfiguredStreamSchema }
