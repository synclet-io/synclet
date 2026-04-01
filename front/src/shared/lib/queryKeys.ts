export const connectorKeys = {
  all: (workspaceId: string) => ['connectors', workspaceId] as const,
  list: (workspaceId: string) => [...connectorKeys.all(workspaceId), 'list'] as const,
  managed: (workspaceId: string, repositoryId?: string | null, search?: string) =>
    [...connectorKeys.all(workspaceId), 'managed', repositoryId ?? '', search ?? ''] as const,
  spec: (workspaceId: string, name: string) => [...connectorKeys.all(workspaceId), 'spec', name] as const,
} as const

export const sourceKeys = {
  all: (workspaceId: string) => ['sources', workspaceId] as const,
  list: (workspaceId: string) => [...sourceKeys.all(workspaceId), 'list'] as const,
  detail: (workspaceId: string, id: string) => [...sourceKeys.all(workspaceId), 'detail', id] as const,
} as const

export const destinationKeys = {
  all: (workspaceId: string) => ['destinations', workspaceId] as const,
  list: (workspaceId: string) => [...destinationKeys.all(workspaceId), 'list'] as const,
  detail: (workspaceId: string, id: string) => [...destinationKeys.all(workspaceId), 'detail', id] as const,
} as const

export const connectionKeys = {
  all: (workspaceId: string) => ['connections', workspaceId] as const,
  list: (workspaceId: string) => [...connectionKeys.all(workspaceId), 'list'] as const,
  detail: (workspaceId: string, id: string) => [...connectionKeys.all(workspaceId), 'detail', id] as const,
  streamStates: (workspaceId: string, connectionId: string) =>
    [...connectionKeys.all(workspaceId), 'streamStates', connectionId] as const,
} as const

export const jobKeys = {
  all: (workspaceId: string) => ['jobs', workspaceId] as const,
  list: (workspaceId: string, connectionId: string) => [...jobKeys.all(workspaceId), 'list', connectionId] as const,
  detail: (workspaceId: string, id: string) => [...jobKeys.all(workspaceId), 'detail', id] as const,
} as const

export const repositoryKeys = {
  all: (workspaceId: string) => ['repositories', workspaceId] as const,
  list: (workspaceId: string) => [...repositoryKeys.all(workspaceId), 'list'] as const,
  connectors: (workspaceId: string, repoId: string, type?: string, filters?: { supportLevel?: string, license?: string, sourceType?: string }) =>
    [...repositoryKeys.all(workspaceId), 'connectors', repoId, type ?? 'all', filters?.supportLevel ?? '', filters?.license ?? '', filters?.sourceType ?? ''] as const,
} as const

export const workspaceKeys = {
  all: ['workspaces'] as const,
  list: () => [...workspaceKeys.all, 'list'] as const,
  detail: (id: string) => [...workspaceKeys.all, 'detail', id] as const,
  members: (id: string) => [...workspaceKeys.all, 'members', id] as const,
  invites: () => [...workspaceKeys.all, 'invites'] as const,
} as const

export const statsKeys = {
  all: (workspaceId: string) => ['stats', workspaceId] as const,
  workspace: (workspaceId: string, timeRange: string) => [...statsKeys.all(workspaceId), 'workspace', timeRange] as const,
  connection: (workspaceId: string, connectionId: string, timeRange: string) => [...statsKeys.all(workspaceId), 'connection', connectionId, timeRange] as const,
  timeline: (workspaceId: string, timeRange: string, connectionId: string) => [...statsKeys.all(workspaceId), 'timeline', timeRange, connectionId] as const,
} as const

export const notificationKeys = {
  all: (workspaceId: string) => ['notifications', workspaceId] as const,
  channels: (workspaceId: string) => [...notificationKeys.all(workspaceId), 'channels'] as const,
  rules: (workspaceId: string, connectionId?: string) => [...notificationKeys.all(workspaceId), 'rules', connectionId ?? 'all'] as const,
} as const

export const inviteKeys = {
  all: ['invite'] as const,
  byToken: (token: string) => [...inviteKeys.all, token] as const,
} as const
