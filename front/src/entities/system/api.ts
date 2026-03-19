import type { SystemInfo } from './types'
import { sourceClient } from '@shared/api/services'
import { WorkspacesMode } from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'

export async function getSystemInfo(): Promise<SystemInfo> {
  const res = await sourceClient.getSystemInfo({})
  return {
    registrationEnabled: res.registrationEnabled,
    workspacesMode: res.workspacesMode === WorkspacesMode.MULTI ? 'multi' : 'single',
  }
}
