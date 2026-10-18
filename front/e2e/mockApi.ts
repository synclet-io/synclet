import type { Page } from '@playwright/test'

export const FIXTURE_ID = '00000000-0000-4000-8000-000000000001'
export const WORKSPACE_ID = '00000000-0000-4000-8000-0000000000aa'

const now = new Date().toISOString()
const hourAgo = new Date(Date.now() - 3600_000).toISOString()

const source = {
  id: FIXTURE_ID,
  workspaceId: WORKSPACE_ID,
  name: 'Production Postgres',
  managedConnectorId: FIXTURE_ID,
  config: { host: 'db.internal', port: 5432 },
  createdAt: hourAgo,
  updatedAt: now,
  connectorName: 'source-postgres',
  connectorImage: 'synclet/source-postgres',
  connectorTag: '1.0.0',
}

const destination = {
  id: FIXTURE_ID,
  workspaceId: WORKSPACE_ID,
  name: 'Warehouse',
  managedConnectorId: FIXTURE_ID,
  config: { bucket: 'warehouse' },
  createdAt: hourAgo,
  updatedAt: now,
}

const connection = {
  id: FIXTURE_ID,
  workspaceId: WORKSPACE_ID,
  name: 'Postgres to Warehouse',
  status: 'CONNECTION_STATUS_ACTIVE',
  sourceId: FIXTURE_ID,
  destinationId: FIXTURE_ID,
  schedule: '0 * * * *',
  schemaChangePolicy: 'SCHEMA_CHANGE_POLICY_PROPAGATE',
  createdAt: hourAgo,
  updatedAt: now,
  maxAttempts: 3,
  namespaceDefinition: 'NAMESPACE_DEFINITION_SOURCE',
  nextScheduledAt: now,
  sourceName: source.name,
  destinationName: destination.name,
  sourceConnectorName: 'source-postgres',
  destinationConnectorName: 'destination-s3',
}

const job = {
  id: FIXTURE_ID,
  connectionId: FIXTURE_ID,
  status: 'JOB_STATUS_COMPLETED',
  jobType: 'JOB_TYPE_SYNC',
  scheduledAt: hourAgo,
  startedAt: hourAgo,
  completedAt: now,
  attempt: 1,
  maxAttempts: 3,
  createdAt: hourAgo,
}

const RPC_PATH = /\/synclet\.publicapi\.[^/]+\/[^/]+$/
const RPC_PREFIX = /^\/synclet\.publicapi\./

const RESPONSES: Record<string, unknown> = {
  'auth.v1.AuthService/GetCurrentUser': {
    user: { id: FIXTURE_ID, email: 'ada@example.com', name: 'Ada Lovelace', createdAt: hourAgo },
    workspaces: [{ workspaceId: WORKSPACE_ID, workspaceName: 'Acme', role: 'MEMBER_ROLE_ADMIN' }],
  },
  'pipeline.v1.SourceService/GetSystemInfo': {
    registrationEnabled: true,
    workspacesMode: 'WORKSPACES_MODE_MULTI',
    emailEnabled: true,
  },
  'pipeline.v1.SourceService/ListExecutors': { executors: [] },
  'pipeline.v1.SourceService/ListSources': { sources: [source], total: 1 },
  'pipeline.v1.SourceService/GetSource': { source },
  'pipeline.v1.DestinationService/ListDestinations': { destinations: [destination], total: 1 },
  'pipeline.v1.DestinationService/GetDestination': { destination },
  'pipeline.v1.ConnectionService/ListConnections': { connections: [connection], total: 1 },
  'pipeline.v1.ConnectionService/GetConnection': { connection },
  'pipeline.v1.JobService/ListJobs': { jobs: [job], total: 1 },
  'pipeline.v1.JobService/GetJob': { job },
  'workspace.v1.WorkspaceService/ListWorkspaces': {
    workspaces: [{ workspace: { id: WORKSPACE_ID, name: 'Acme' }, role: 'MEMBER_ROLE_ADMIN' }],
  },
}

// Answers every Connect RPC with a fixture, or an empty message for methods without one.
export async function mockApi(page: Page) {
  await page.route(RPC_PATH, async (route) => {
    const method = new URL(route.request().url()).pathname.replace(RPC_PREFIX, '')
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(RESPONSES[method] ?? {}),
    })
  })
}
