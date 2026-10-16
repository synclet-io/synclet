import type { ExecutorKind, RuntimeDefaults, SystemInfo } from './types'
import type { RuntimeDefaults as RuntimeDefaultsProto } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'
import { sourceClient } from '@shared/api/services'
import { ExecutorType, WorkspacesMode } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'

function executorKind(executor: ExecutorType | undefined): ExecutorKind {
  switch (executor) {
    case ExecutorType.DOCKER:
      return 'docker'
    case ExecutorType.K8S:
      return 'k8s'
    default:
      return 'external'
  }
}

function toRuntimeDefaults(rd: RuntimeDefaultsProto | undefined): RuntimeDefaults {
  return {
    executor: executorKind(rd?.executor),
    cpuRequest: rd?.cpuRequest ?? '',
    cpuLimit: rd?.cpuLimit ?? '',
    memoryRequest: rd?.memoryRequest ?? '',
    memoryLimit: rd?.memoryLimit ?? '',
    serviceAccountName: rd?.serviceAccountName ?? '',
    maxConcurrentJobs: rd?.maxConcurrentJobs ?? 0,
    idleTimeoutSeconds: Number(rd?.idleTimeoutSeconds ?? 0),
    logArchiveBackend: rd?.logArchiveBackend ?? '',
    dockerMaxConcurrentJobs: rd?.dockerMaxConcurrentJobs ?? 0,
    cpuExecutorDefault: rd?.cpuExecutorDefault ?? false,
    memoryExecutorDefault: rd?.memoryExecutorDefault ?? false,
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
