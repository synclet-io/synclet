package pipelineservice

import (
	strconv "strconv"
	// user code 'imports'
	// end user code 'imports'
)

type SchemaChangePolicy byte

const (
	SchemaChangePolicyPropagate SchemaChangePolicy = iota + 1
	SchemaChangePolicyIgnore
	SchemaChangePolicyPause
)

// user code 'SchemaChangePolicy methods'
// end user code 'SchemaChangePolicy methods'
func (s SchemaChangePolicy) IsValid() bool {
	return s > 0 && s < 4
}
func (s SchemaChangePolicy) IsPropagate() bool {
	return s == SchemaChangePolicyPropagate
}
func (s SchemaChangePolicy) IsIgnore() bool {
	return s == SchemaChangePolicyIgnore
}
func (s SchemaChangePolicy) IsPause() bool {
	return s == SchemaChangePolicyPause
}
func (s SchemaChangePolicy) String() string {
	const names = "PropagateIgnorePause"

	var indexes = [...]int32{0, 9, 15, 20}
	if s < 1 || s > 3 {
		return "SchemaChangePolicy(" + strconv.FormatInt(int64(s), 10) + ")"
	}

	return names[indexes[s-1]:indexes[s]]
}

type ConnectionStatus byte

const (
	ConnectionStatusActive ConnectionStatus = iota + 1
	ConnectionStatusInactive
	ConnectionStatusPaused
)

// user code 'ConnectionStatus methods'
// end user code 'ConnectionStatus methods'
func (c ConnectionStatus) IsValid() bool {
	return c > 0 && c < 4
}
func (c ConnectionStatus) IsActive() bool {
	return c == ConnectionStatusActive
}
func (c ConnectionStatus) IsInactive() bool {
	return c == ConnectionStatusInactive
}
func (c ConnectionStatus) IsPaused() bool {
	return c == ConnectionStatusPaused
}
func (c ConnectionStatus) String() string {
	const names = "ActiveInactivePaused"

	var indexes = [...]int32{0, 6, 14, 20}
	if c < 1 || c > 3 {
		return "ConnectionStatus(" + strconv.FormatInt(int64(c), 10) + ")"
	}

	return names[indexes[c-1]:indexes[c]]
}

type JobStatus byte

const (
	JobStatusScheduled JobStatus = iota + 1
	JobStatusStarting
	JobStatusRunning
	JobStatusCompleted
	JobStatusFailed
	JobStatusCancelled
)

// user code 'JobStatus methods'
// end user code 'JobStatus methods'
func (j JobStatus) IsValid() bool {
	return j > 0 && j < 7
}
func (j JobStatus) IsScheduled() bool {
	return j == JobStatusScheduled
}
func (j JobStatus) IsStarting() bool {
	return j == JobStatusStarting
}
func (j JobStatus) IsRunning() bool {
	return j == JobStatusRunning
}
func (j JobStatus) IsCompleted() bool {
	return j == JobStatusCompleted
}
func (j JobStatus) IsFailed() bool {
	return j == JobStatusFailed
}
func (j JobStatus) IsCancelled() bool {
	return j == JobStatusCancelled
}
func (j JobStatus) String() string {
	const names = "ScheduledStartingRunningCompletedFailedCancelled"

	var indexes = [...]int32{0, 9, 17, 24, 33, 39, 48}
	if j < 1 || j > 6 {
		return "JobStatus(" + strconv.FormatInt(int64(j), 10) + ")"
	}

	return names[indexes[j-1]:indexes[j]]
}

type JobType byte

const (
	JobTypeSync JobType = iota + 1
	JobTypeDiscover
	JobTypeCheck
)

// user code 'JobType methods'
// end user code 'JobType methods'
func (j JobType) IsValid() bool {
	return j > 0 && j < 4
}
func (j JobType) IsSync() bool {
	return j == JobTypeSync
}
func (j JobType) IsDiscover() bool {
	return j == JobTypeDiscover
}
func (j JobType) IsCheck() bool {
	return j == JobTypeCheck
}
func (j JobType) String() string {
	const names = "SyncDiscoverCheck"

	var indexes = [...]int32{0, 4, 12, 17}
	if j < 1 || j > 3 {
		return "JobType(" + strconv.FormatInt(int64(j), 10) + ")"
	}

	return names[indexes[j-1]:indexes[j]]
}

type NamespaceDefinition byte

const (
	NamespaceDefinitionSource NamespaceDefinition = iota + 1
	NamespaceDefinitionDestination
	NamespaceDefinitionCustom
)

// user code 'NamespaceDefinition methods'
// end user code 'NamespaceDefinition methods'
func (n NamespaceDefinition) IsValid() bool {
	return n > 0 && n < 4
}
func (n NamespaceDefinition) IsSource() bool {
	return n == NamespaceDefinitionSource
}
func (n NamespaceDefinition) IsDestination() bool {
	return n == NamespaceDefinitionDestination
}
func (n NamespaceDefinition) IsCustom() bool {
	return n == NamespaceDefinitionCustom
}
func (n NamespaceDefinition) String() string {
	const names = "SourceDestinationCustom"

	var indexes = [...]int32{0, 6, 17, 23}
	if n < 1 || n > 3 {
		return "NamespaceDefinition(" + strconv.FormatInt(int64(n), 10) + ")"
	}

	return names[indexes[n-1]:indexes[n]]
}

type RepositoryStatus byte

const (
	RepositoryStatusSyncing RepositoryStatus = iota + 1
	RepositoryStatusSynced
	RepositoryStatusFailed
)

// user code 'RepositoryStatus methods'
// end user code 'RepositoryStatus methods'
func (r RepositoryStatus) IsValid() bool {
	return r > 0 && r < 4
}
func (r RepositoryStatus) IsSyncing() bool {
	return r == RepositoryStatusSyncing
}
func (r RepositoryStatus) IsSynced() bool {
	return r == RepositoryStatusSynced
}
func (r RepositoryStatus) IsFailed() bool {
	return r == RepositoryStatusFailed
}
func (r RepositoryStatus) String() string {
	const names = "SyncingSyncedFailed"

	var indexes = [...]int32{0, 7, 13, 19}
	if r < 1 || r > 3 {
		return "RepositoryStatus(" + strconv.FormatInt(int64(r), 10) + ")"
	}

	return names[indexes[r-1]:indexes[r]]
}

type ConnectorTaskType byte

const (
	ConnectorTaskTypeCheck ConnectorTaskType = iota + 1
	ConnectorTaskTypeSpec
	ConnectorTaskTypeDiscover
)

// user code 'ConnectorTaskType methods'
// end user code 'ConnectorTaskType methods'
func (c ConnectorTaskType) IsValid() bool {
	return c > 0 && c < 4
}
func (c ConnectorTaskType) IsCheck() bool {
	return c == ConnectorTaskTypeCheck
}
func (c ConnectorTaskType) IsSpec() bool {
	return c == ConnectorTaskTypeSpec
}
func (c ConnectorTaskType) IsDiscover() bool {
	return c == ConnectorTaskTypeDiscover
}
func (c ConnectorTaskType) String() string {
	const names = "CheckSpecDiscover"

	var indexes = [...]int32{0, 5, 9, 17}
	if c < 1 || c > 3 {
		return "ConnectorTaskType(" + strconv.FormatInt(int64(c), 10) + ")"
	}

	return names[indexes[c-1]:indexes[c]]
}

type ConnectorTaskStatus byte

const (
	ConnectorTaskStatusPending ConnectorTaskStatus = iota + 1
	ConnectorTaskStatusRunning
	ConnectorTaskStatusCompleted
	ConnectorTaskStatusFailed
)

// user code 'ConnectorTaskStatus methods'
// end user code 'ConnectorTaskStatus methods'
func (c ConnectorTaskStatus) IsValid() bool {
	return c > 0 && c < 5
}
func (c ConnectorTaskStatus) IsPending() bool {
	return c == ConnectorTaskStatusPending
}
func (c ConnectorTaskStatus) IsRunning() bool {
	return c == ConnectorTaskStatusRunning
}
func (c ConnectorTaskStatus) IsCompleted() bool {
	return c == ConnectorTaskStatusCompleted
}
func (c ConnectorTaskStatus) IsFailed() bool {
	return c == ConnectorTaskStatusFailed
}
func (c ConnectorTaskStatus) String() string {
	const names = "PendingRunningCompletedFailed"

	var indexes = [...]int32{0, 7, 14, 23, 29}
	if c < 1 || c > 4 {
		return "ConnectorTaskStatus(" + strconv.FormatInt(int64(c), 10) + ")"
	}

	return names[indexes[c-1]:indexes[c]]
}

type ConnectorType byte

const (
	ConnectorTypeSource ConnectorType = iota + 1
	ConnectorTypeDestination
)

// user code 'ConnectorType methods'
// end user code 'ConnectorType methods'
func (c ConnectorType) IsValid() bool {
	return c > 0 && c < 3
}
func (c ConnectorType) IsSource() bool {
	return c == ConnectorTypeSource
}
func (c ConnectorType) IsDestination() bool {
	return c == ConnectorTypeDestination
}
func (c ConnectorType) String() string {
	const names = "SourceDestination"

	var indexes = [...]int32{0, 6, 17}
	if c < 1 || c > 2 {
		return "ConnectorType(" + strconv.FormatInt(int64(c), 10) + ")"
	}

	return names[indexes[c-1]:indexes[c]]
}

type SupportLevel byte

const (
	SupportLevelCommunity SupportLevel = iota + 1
	SupportLevelCertified
	SupportLevelUnknown
)

// user code 'SupportLevel methods'
// end user code 'SupportLevel methods'
func (s SupportLevel) IsValid() bool {
	return s > 0 && s < 4
}
func (s SupportLevel) IsCommunity() bool {
	return s == SupportLevelCommunity
}
func (s SupportLevel) IsCertified() bool {
	return s == SupportLevelCertified
}
func (s SupportLevel) IsUnknown() bool {
	return s == SupportLevelUnknown
}
func (s SupportLevel) String() string {
	const names = "CommunityCertifiedUnknown"

	var indexes = [...]int32{0, 9, 18, 25}
	if s < 1 || s > 3 {
		return "SupportLevel(" + strconv.FormatInt(int64(s), 10) + ")"
	}

	return names[indexes[s-1]:indexes[s]]
}

type SourceType byte

const (
	SourceTypeAPI SourceType = iota + 1
	SourceTypeDatabase
	SourceTypeFile
	SourceTypeUnknown
)

// user code 'SourceType methods'
// end user code 'SourceType methods'
func (s SourceType) IsValid() bool {
	return s > 0 && s < 5
}
func (s SourceType) IsAPI() bool {
	return s == SourceTypeAPI
}
func (s SourceType) IsDatabase() bool {
	return s == SourceTypeDatabase
}
func (s SourceType) IsFile() bool {
	return s == SourceTypeFile
}
func (s SourceType) IsUnknown() bool {
	return s == SourceTypeUnknown
}
func (s SourceType) String() string {
	const names = "APIDatabaseFileUnknown"

	var indexes = [...]int32{0, 3, 11, 15, 22}
	if s < 1 || s > 4 {
		return "SourceType(" + strconv.FormatInt(int64(s), 10) + ")"
	}

	return names[indexes[s-1]:indexes[s]]
}

type ReleaseStage byte

const (
	ReleaseStageGenerallyAvailable ReleaseStage = iota + 1
	ReleaseStageBeta
	ReleaseStageAlpha
	ReleaseStageCustom
	ReleaseStageUnknown
)

// user code 'ReleaseStage methods'
// end user code 'ReleaseStage methods'
func (r ReleaseStage) IsValid() bool {
	return r > 0 && r < 6
}
func (r ReleaseStage) IsGenerallyAvailable() bool {
	return r == ReleaseStageGenerallyAvailable
}
func (r ReleaseStage) IsBeta() bool {
	return r == ReleaseStageBeta
}
func (r ReleaseStage) IsAlpha() bool {
	return r == ReleaseStageAlpha
}
func (r ReleaseStage) IsCustom() bool {
	return r == ReleaseStageCustom
}
func (r ReleaseStage) IsUnknown() bool {
	return r == ReleaseStageUnknown
}
func (r ReleaseStage) String() string {
	const names = "GenerallyAvailableBetaAlphaCustomUnknown"

	var indexes = [...]int32{0, 18, 22, 27, 33, 40}
	if r < 1 || r > 5 {
		return "ReleaseStage(" + strconv.FormatInt(int64(r), 10) + ")"
	}

	return names[indexes[r-1]:indexes[r]]
}

type BucketSize byte

const (
	BucketSizeHourly BucketSize = iota + 1
	BucketSizeDaily
)

// user code 'BucketSize methods'
// end user code 'BucketSize methods'
func (b BucketSize) IsValid() bool {
	return b > 0 && b < 3
}
func (b BucketSize) IsHourly() bool {
	return b == BucketSizeHourly
}
func (b BucketSize) IsDaily() bool {
	return b == BucketSizeDaily
}
func (b BucketSize) String() string {
	const names = "HourlyDaily"

	var indexes = [...]int32{0, 6, 11}
	if b < 1 || b > 2 {
		return "BucketSize(" + strconv.FormatInt(int64(b), 10) + ")"
	}

	return names[indexes[b-1]:indexes[b]]
}

type Health byte

const (
	HealthHealthy Health = iota + 1
	HealthWarning
	HealthFailing
	HealthDisabled
)

// user code 'Health methods'
// end user code 'Health methods'
func (h Health) IsValid() bool {
	return h > 0 && h < 5
}
func (h Health) IsHealthy() bool {
	return h == HealthHealthy
}
func (h Health) IsWarning() bool {
	return h == HealthWarning
}
func (h Health) IsFailing() bool {
	return h == HealthFailing
}
func (h Health) IsDisabled() bool {
	return h == HealthDisabled
}
func (h Health) String() string {
	const names = "HealthyWarningFailingDisabled"

	var indexes = [...]int32{0, 7, 14, 21, 29}
	if h < 1 || h > 4 {
		return "Health(" + strconv.FormatInt(int64(h), 10) + ")"
	}

	return names[indexes[h-1]:indexes[h]]
}

type FailureCategory byte

const (
	FailureCategoryTimeout FailureCategory = iota + 1
	FailureCategoryOOM
	FailureCategoryConnector
	FailureCategoryInfrastructure
	FailureCategoryUnknown
)

// user code 'FailureCategory methods'
// end user code 'FailureCategory methods'
func (f FailureCategory) IsValid() bool {
	return f > 0 && f < 6
}
func (f FailureCategory) IsTimeout() bool {
	return f == FailureCategoryTimeout
}
func (f FailureCategory) IsOOM() bool {
	return f == FailureCategoryOOM
}
func (f FailureCategory) IsConnector() bool {
	return f == FailureCategoryConnector
}
func (f FailureCategory) IsInfrastructure() bool {
	return f == FailureCategoryInfrastructure
}
func (f FailureCategory) IsUnknown() bool {
	return f == FailureCategoryUnknown
}
func (f FailureCategory) String() string {
	const names = "TimeoutOOMConnectorInfrastructureUnknown"

	var indexes = [...]int32{0, 7, 10, 19, 33, 40}
	if f < 1 || f > 5 {
		return "FailureCategory(" + strconv.FormatInt(int64(f), 10) + ")"
	}

	return names[indexes[f-1]:indexes[f]]
}

type SyncStatus byte

const (
	SyncStatusCompleted SyncStatus = iota + 1
	SyncStatusFailed
)

// user code 'SyncStatus methods'
// end user code 'SyncStatus methods'
func (s SyncStatus) IsValid() bool {
	return s > 0 && s < 3
}
func (s SyncStatus) IsCompleted() bool {
	return s == SyncStatusCompleted
}
func (s SyncStatus) IsFailed() bool {
	return s == SyncStatusFailed
}
func (s SyncStatus) String() string {
	const names = "CompletedFailed"

	var indexes = [...]int32{0, 9, 15}
	if s < 1 || s > 2 {
		return "SyncStatus(" + strconv.FormatInt(int64(s), 10) + ")"
	}

	return names[indexes[s-1]:indexes[s]]
}
