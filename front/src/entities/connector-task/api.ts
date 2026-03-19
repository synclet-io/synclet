import type { ConnectorTaskResult, ConnectorTaskStatus, ConnectorTaskType } from './types'
import { connectorTaskClient } from '@shared/api/services'
import {
  ConnectorTaskStatus as ProtoTaskStatus,
  ConnectorTaskType as ProtoTaskType,
} from '@/gen/synclet/publicapi/pipeline/v1/pipeline_pb'

function mapStatus(protoStatus: ProtoTaskStatus): ConnectorTaskStatus {
  switch (protoStatus) {
    case ProtoTaskStatus.PENDING: return 'pending'
    case ProtoTaskStatus.RUNNING: return 'running'
    case ProtoTaskStatus.COMPLETED: return 'completed'
    case ProtoTaskStatus.FAILED: return 'failed'
    default: return 'pending'
  }
}

function mapTaskType(protoType: ProtoTaskType): ConnectorTaskType {
  switch (protoType) {
    case ProtoTaskType.CHECK: return 'check'
    case ProtoTaskType.SPEC: return 'spec'
    case ProtoTaskType.DISCOVER: return 'discover'
    default: return 'check'
  }
}

export async function getConnectorTaskResult(taskId: string): Promise<ConnectorTaskResult> {
  const res = await connectorTaskClient.getConnectorTaskResult({ taskId })

  const result: ConnectorTaskResult = {
    status: mapStatus(res.status),
    taskType: mapTaskType(res.taskType),
    errorMessage: res.errorMessage || '',
  }

  if (res.result.case === 'checkResult' && res.result.value) {
    result.checkResult = {
      success: res.result.value.success,
      message: res.result.value.message || '',
    }
  }
  else if (res.result.case === 'specResult' && res.result.value) {
    result.specResult = {
      spec: (res.result.value.spec ?? {}) as Record<string, unknown>,
    }
  }
  else if (res.result.case === 'discoverResult' && res.result.value) {
    result.discoverResult = {
      catalog: (res.result.value.catalog ?? {}) as Record<string, unknown>,
    }
  }

  return result
}
