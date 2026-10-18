import type { ExecutorInstance, ExecutorKind, ExecutorResourceDefaults, RuntimeDefaults, SystemInfo } from './types'
import type {
  ExecutorResourceDefaults as ExecutorResourceDefaultsProto,
  RuntimeDefaults as RuntimeDefaultsProto,
} from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { sourceClient } from '@shared/api/services'
import { tsToDate } from '@shared/lib/formatting'
import { ExecutorType, WorkspacesMode } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'

function executorKind(type: ExecutorType): ExecutorKind {
  switch (type) {
    case ExecutorType.DOCKER:
      return 'docker'
    case ExecutorType.K8S:
      return 'k8s'
    default:
      return 'unknown'
  }
}

function toResourceDefaults(rd: ExecutorResourceDefaultsProto | undefined): ExecutorResourceDefaults {
  return {
    cpuRequest: rd?.cpuRequest ?? '',
    cpuLimit: rd?.cpuLimit ?? '',
    memoryRequest: rd?.memoryRequest ?? '',
    memoryLimit: rd?.memoryLimit ?? '',
    cpuExecutorDefault: rd?.cpuExecutorDefault ?? false,
    memoryExecutorDefault: rd?.memoryExecutorDefault ?? false,
  }
}

function toRuntimeDefaults(rd: RuntimeDefaultsProto | undefined): RuntimeDefaults {
  return {
    cpuRequest: rd?.cpuRequest ?? '',
    cpuLimit: rd?.cpuLimit ?? '',
    memoryRequest: rd?.memoryRequest ?? '',
    memoryLimit: rd?.memoryLimit ?? '',
    serviceAccountName: rd?.serviceAccountName ?? '',
    maxConcurrentJobs: rd?.maxConcurrentJobs ?? 0,
    idleTimeoutSeconds: Number(rd?.idleTimeoutSeconds ?? 0),
    logArchiveBackend: rd?.logArchiveBackend ?? '',
    executors: (rd?.executors ?? []).map(e => ({
      type: executorKind(e.type),
      capacity: e.capacity,
      resourceDefaults: toResourceDefaults(e.resourceDefaults),
    })),
  }
}

export async function getSystemInfo(): Promise<SystemInfo> {
  const res = await sourceClient.getSystemInfo({})
  return {
    registrationEnabled: res.registrationEnabled,
    workspacesMode: res.workspacesMode === WorkspacesMode.MULTI ? 'multi' : 'single',
    emailEnabled: res.emailEnabled,
  }
}

export async function getRuntimeDefaults(): Promise<RuntimeDefaults> {
  const res = await sourceClient.getRuntimeDefaults({})
  return toRuntimeDefaults(res.runtimeDefaults)
}

export async function listExecutors(): Promise<ExecutorInstance[]> {
  const res = await sourceClient.listExecutors({})
  return res.executors.map(e => ({
    id: e.id,
    type: executorKind(e.type),
    version: e.version,
    capacity: e.capacity,
    nativeDestinations: [...e.nativeDestinations],
    resourceDefaults: toResourceDefaults(e.resourceDefaults),
    firstSeenAt: tsToDate(e.firstSeenAt),
    lastSeenAt: tsToDate(e.lastSeenAt),
  }))
}
