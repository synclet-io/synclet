export type ConnectorTaskStatus = 'pending' | 'running' | 'completed' | 'failed'

export type ConnectorTaskType = 'check' | 'spec' | 'discover'

export interface ConnectorTaskResult {
  status: ConnectorTaskStatus
  taskType: ConnectorTaskType
  errorMessage: string
  checkResult?: { success: boolean, message: string }
  specResult?: { spec: Record<string, unknown> }
  discoverResult?: { catalog: Record<string, unknown> }
}
