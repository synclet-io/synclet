package pipelineservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

type ConnectorTaskPayload interface {
	isConnectorTaskPayload()
	ConnectorTaskPayloadEquals(ConnectorTaskPayload) bool
	// user code 'ConnectorTaskPayload methods'
	// end user code 'ConnectorTaskPayload methods'
}

func (*CheckPayload) isConnectorTaskPayload() {}
func (c *CheckPayload) ConnectorTaskPayloadEquals(to ConnectorTaskPayload) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*CheckPayload)
	if !ok {
		return false
	}

	return c.Equals(toTyped)
}
func (*SpecPayload) isConnectorTaskPayload() {}
func (s *SpecPayload) ConnectorTaskPayloadEquals(to ConnectorTaskPayload) bool {
	if (s == nil) != (to == nil) {
		return false
	}
	if s == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*SpecPayload)
	if !ok {
		return false
	}

	return s.Equals(toTyped)
}
func (*DiscoverPayload) isConnectorTaskPayload() {}
func (d *DiscoverPayload) ConnectorTaskPayloadEquals(to ConnectorTaskPayload) bool {
	if (d == nil) != (to == nil) {
		return false
	}
	if d == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*DiscoverPayload)
	if !ok {
		return false
	}

	return d.Equals(toTyped)
}

func copyConnectorTaskPayload(val ConnectorTaskPayload) ConnectorTaskPayload {
	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case *CheckPayload:
		valCopy := val.Copy()
		return &valCopy
	case *SpecPayload:
		valCopy := val.Copy()
		return &valCopy
	case *DiscoverPayload:
		valCopy := val.Copy()
		return &valCopy
	}
	panic("called copyConnectorTaskPayload with invalid type")
}

type ConnectorTaskResult interface {
	isConnectorTaskResult()
	ConnectorTaskResultEquals(ConnectorTaskResult) bool
	// user code 'ConnectorTaskResult methods'
	// end user code 'ConnectorTaskResult methods'
}

func (*CheckResult) isConnectorTaskResult() {}
func (c *CheckResult) ConnectorTaskResultEquals(to ConnectorTaskResult) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*CheckResult)
	if !ok {
		return false
	}

	return c.Equals(toTyped)
}
func (*SpecResult) isConnectorTaskResult() {}
func (s *SpecResult) ConnectorTaskResultEquals(to ConnectorTaskResult) bool {
	if (s == nil) != (to == nil) {
		return false
	}
	if s == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*SpecResult)
	if !ok {
		return false
	}

	return s.Equals(toTyped)
}
func (*DiscoverResult) isConnectorTaskResult() {}
func (d *DiscoverResult) ConnectorTaskResultEquals(to ConnectorTaskResult) bool {
	if (d == nil) != (to == nil) {
		return false
	}
	if d == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*DiscoverResult)
	if !ok {
		return false
	}

	return d.Equals(toTyped)
}

func copyConnectorTaskResult(val ConnectorTaskResult) ConnectorTaskResult {
	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case *CheckResult:
		valCopy := val.Copy()
		return &valCopy
	case *SpecResult:
		valCopy := val.Copy()
		return &valCopy
	case *DiscoverResult:
		valCopy := val.Copy()
		return &valCopy
	}
	panic("called copyConnectorTaskResult with invalid type")
}

type ManagedConnectorField byte

const (
	ManagedConnectorFieldID ManagedConnectorField = iota + 1
	ManagedConnectorFieldWorkspaceID
	ManagedConnectorFieldDockerImage
	ManagedConnectorFieldDockerTag
	ManagedConnectorFieldName
	ManagedConnectorFieldConnectorType
	ManagedConnectorFieldSpec
	ManagedConnectorFieldCreatedAt
	ManagedConnectorFieldUpdatedAt
	ManagedConnectorFieldRepositoryID
)

type ManagedConnectorFilter struct {
	ID            filter.Filter[uuid.UUID]
	WorkspaceID   filter.Filter[uuid.UUID]
	DockerImage   filter.Filter[string]
	Name          filter.Filter[string]
	ConnectorType filter.Filter[ConnectorType]
	RepositoryID  filter.Filter[*uuid.UUID]
	Or            []*ManagedConnectorFilter
	And           []*ManagedConnectorFilter
}
type ManagedConnectorOrder order.Order[ManagedConnectorField]

type ManagedConnector struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	DockerImage   string
	DockerTag     string
	Name          string
	ConnectorType ConnectorType
	Spec          jsonb
	CreatedAt     time.Time
	UpdatedAt     time.Time
	RepositoryID  *uuid.UUID
}

// user code 'ManagedConnector methods'
// end user code 'ManagedConnector methods'

func (m *ManagedConnector) Copy() ManagedConnector {
	var result ManagedConnector
	result.ID = m.ID
	result.WorkspaceID = m.WorkspaceID
	result.DockerImage = m.DockerImage
	result.DockerTag = m.DockerTag
	result.Name = m.Name
	result.ConnectorType = m.ConnectorType // enum
	result.Spec = m.Spec
	result.CreatedAt = m.CreatedAt
	result.UpdatedAt = m.UpdatedAt
	if m.RepositoryID != nil {
		var tmp uuid.UUID
		tmp = (*m.RepositoryID)
		result.RepositoryID = &tmp
	}

	return result
}
func (m *ManagedConnector) Equals(to *ManagedConnector) bool {
	if (m == nil) != (to == nil) {
		return false
	}
	if m == nil && to == nil {
		return true
	}
	if m.ID != to.ID {
		return false
	}
	if m.WorkspaceID != to.WorkspaceID {
		return false
	}
	if m.DockerImage != to.DockerImage {
		return false
	}
	if m.DockerTag != to.DockerTag {
		return false
	}
	if m.Name != to.Name {
		return false
	}
	if m.ConnectorType != to.ConnectorType {
		return false
	}
	if m.Spec != to.Spec {
		return false
	}
	if m.CreatedAt != to.CreatedAt {
		return false
	}
	if m.UpdatedAt != to.UpdatedAt {
		return false
	}
	if (m.RepositoryID == nil) != (to.RepositoryID == nil) {
		return false
	}
	if m.RepositoryID != nil && to.RepositoryID != nil {
		if (*m.RepositoryID) != (*to.RepositoryID) {
			return false
		}
	}

	return true
}

type RepositoryField byte

const (
	RepositoryFieldID RepositoryField = iota + 1
	RepositoryFieldWorkspaceID
	RepositoryFieldName
	RepositoryFieldURL
	RepositoryFieldAuthHeader
	RepositoryFieldStatus
	RepositoryFieldLastSyncedAt
	RepositoryFieldConnectorCount
	RepositoryFieldLastError
	RepositoryFieldCreatedAt
	RepositoryFieldUpdatedAt
)

type RepositoryFilter struct {
	ID          filter.Filter[uuid.UUID]
	WorkspaceID filter.Filter[uuid.UUID]
	Status      filter.Filter[RepositoryStatus]
	Or          []*RepositoryFilter
	And         []*RepositoryFilter
}
type RepositoryOrder order.Order[RepositoryField]

type Repository struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	Name           string
	URL            string
	AuthHeader     *string
	Status         RepositoryStatus
	LastSyncedAt   *time.Time
	ConnectorCount int
	LastError      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// user code 'Repository methods'
// end user code 'Repository methods'

func (r *Repository) Copy() Repository {
	var result Repository
	result.ID = r.ID
	result.WorkspaceID = r.WorkspaceID
	result.Name = r.Name
	result.URL = r.URL
	if r.AuthHeader != nil {
		var tmp string
		tmp = (*r.AuthHeader)
		result.AuthHeader = &tmp
	}
	result.Status = r.Status // enum
	if r.LastSyncedAt != nil {
		var tmp1 time.Time
		tmp1 = (*r.LastSyncedAt)
		result.LastSyncedAt = &tmp1
	}
	result.ConnectorCount = r.ConnectorCount
	if r.LastError != nil {
		var tmp2 string
		tmp2 = (*r.LastError)
		result.LastError = &tmp2
	}
	result.CreatedAt = r.CreatedAt
	result.UpdatedAt = r.UpdatedAt

	return result
}
func (r *Repository) Equals(to *Repository) bool {
	if (r == nil) != (to == nil) {
		return false
	}
	if r == nil && to == nil {
		return true
	}
	if r.ID != to.ID {
		return false
	}
	if r.WorkspaceID != to.WorkspaceID {
		return false
	}
	if r.Name != to.Name {
		return false
	}
	if r.URL != to.URL {
		return false
	}
	if (r.AuthHeader == nil) != (to.AuthHeader == nil) {
		return false
	}
	if r.AuthHeader != nil && to.AuthHeader != nil {
		if (*r.AuthHeader) != (*to.AuthHeader) {
			return false
		}
	}
	if r.Status != to.Status {
		return false
	}
	if (r.LastSyncedAt == nil) != (to.LastSyncedAt == nil) {
		return false
	}
	if r.LastSyncedAt != nil && to.LastSyncedAt != nil {
		if (*r.LastSyncedAt) != (*to.LastSyncedAt) {
			return false
		}
	}
	if r.ConnectorCount != to.ConnectorCount {
		return false
	}
	if (r.LastError == nil) != (to.LastError == nil) {
		return false
	}
	if r.LastError != nil && to.LastError != nil {
		if (*r.LastError) != (*to.LastError) {
			return false
		}
	}
	if r.CreatedAt != to.CreatedAt {
		return false
	}
	if r.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type RepositoryConnectorField byte

const (
	RepositoryConnectorFieldID RepositoryConnectorField = iota + 1
	RepositoryConnectorFieldRepositoryID
	RepositoryConnectorFieldDockerRepository
	RepositoryConnectorFieldDockerImageTag
	RepositoryConnectorFieldName
	RepositoryConnectorFieldConnectorType
	RepositoryConnectorFieldDocumentationURL
	RepositoryConnectorFieldReleaseStage
	RepositoryConnectorFieldIconURL
	RepositoryConnectorFieldSpec
	RepositoryConnectorFieldSupportLevel
	RepositoryConnectorFieldLicense
	RepositoryConnectorFieldSourceType
	RepositoryConnectorFieldMetadata
)

type RepositoryConnectorFilter struct {
	ID               filter.Filter[uuid.UUID]
	RepositoryID     filter.Filter[uuid.UUID]
	DockerRepository filter.Filter[string]
	Name             filter.Filter[string]
	ConnectorType    filter.Filter[ConnectorType]
	SupportLevel     filter.Filter[SupportLevel]
	License          filter.Filter[string]
	SourceType       filter.Filter[SourceType]
	Or               []*RepositoryConnectorFilter
	And              []*RepositoryConnectorFilter
}
type RepositoryConnectorOrder order.Order[RepositoryConnectorField]

type RepositoryConnector struct {
	ID               uuid.UUID
	RepositoryID     uuid.UUID
	DockerRepository string
	DockerImageTag   string
	Name             string
	ConnectorType    ConnectorType
	DocumentationURL string
	ReleaseStage     ReleaseStage
	IconURL          string
	Spec             jsonb
	SupportLevel     SupportLevel
	License          string
	SourceType       SourceType
	Metadata         jsonb
}

// user code 'RepositoryConnector methods'
// end user code 'RepositoryConnector methods'

func (r *RepositoryConnector) Copy() RepositoryConnector {
	var result RepositoryConnector
	result.ID = r.ID
	result.RepositoryID = r.RepositoryID
	result.DockerRepository = r.DockerRepository
	result.DockerImageTag = r.DockerImageTag
	result.Name = r.Name
	result.ConnectorType = r.ConnectorType // enum
	result.DocumentationURL = r.DocumentationURL
	result.ReleaseStage = r.ReleaseStage // enum
	result.IconURL = r.IconURL
	result.Spec = r.Spec
	result.SupportLevel = r.SupportLevel // enum
	result.License = r.License
	result.SourceType = r.SourceType // enum
	result.Metadata = r.Metadata

	return result
}
func (r *RepositoryConnector) Equals(to *RepositoryConnector) bool {
	if (r == nil) != (to == nil) {
		return false
	}
	if r == nil && to == nil {
		return true
	}
	if r.ID != to.ID {
		return false
	}
	if r.RepositoryID != to.RepositoryID {
		return false
	}
	if r.DockerRepository != to.DockerRepository {
		return false
	}
	if r.DockerImageTag != to.DockerImageTag {
		return false
	}
	if r.Name != to.Name {
		return false
	}
	if r.ConnectorType != to.ConnectorType {
		return false
	}
	if r.DocumentationURL != to.DocumentationURL {
		return false
	}
	if r.ReleaseStage != to.ReleaseStage {
		return false
	}
	if r.IconURL != to.IconURL {
		return false
	}
	if r.Spec != to.Spec {
		return false
	}
	if r.SupportLevel != to.SupportLevel {
		return false
	}
	if r.License != to.License {
		return false
	}
	if r.SourceType != to.SourceType {
		return false
	}
	if r.Metadata != to.Metadata {
		return false
	}

	return true
}

type SourceField byte

const (
	SourceFieldID SourceField = iota + 1
	SourceFieldWorkspaceID
	SourceFieldName
	SourceFieldManagedConnectorID
	SourceFieldConfig
	SourceFieldCreatedAt
	SourceFieldUpdatedAt
	SourceFieldRuntimeConfig
)

type SourceFilter struct {
	ID                 filter.Filter[uuid.UUID]
	WorkspaceID        filter.Filter[uuid.UUID]
	Name               filter.Filter[string]
	ManagedConnectorID filter.Filter[uuid.UUID]
	Or                 []*SourceFilter
	And                []*SourceFilter
}
type SourceOrder order.Order[SourceField]

type Source struct {
	ID                 uuid.UUID
	WorkspaceID        uuid.UUID
	Name               string
	ManagedConnectorID uuid.UUID
	Config             jsonb
	CreatedAt          time.Time
	UpdatedAt          time.Time
	RuntimeConfig      *jsonb
}

// user code 'Source methods'
// end user code 'Source methods'

func (s *Source) Copy() Source {
	var result Source
	result.ID = s.ID
	result.WorkspaceID = s.WorkspaceID
	result.Name = s.Name
	result.ManagedConnectorID = s.ManagedConnectorID
	result.Config = s.Config
	result.CreatedAt = s.CreatedAt
	result.UpdatedAt = s.UpdatedAt
	if s.RuntimeConfig != nil {
		var tmp jsonb
		tmp = (*s.RuntimeConfig)
		result.RuntimeConfig = &tmp
	}

	return result
}
func (s *Source) Equals(to *Source) bool {
	if (s == nil) != (to == nil) {
		return false
	}
	if s == nil && to == nil {
		return true
	}
	if s.ID != to.ID {
		return false
	}
	if s.WorkspaceID != to.WorkspaceID {
		return false
	}
	if s.Name != to.Name {
		return false
	}
	if s.ManagedConnectorID != to.ManagedConnectorID {
		return false
	}
	if s.Config != to.Config {
		return false
	}
	if s.CreatedAt != to.CreatedAt {
		return false
	}
	if s.UpdatedAt != to.UpdatedAt {
		return false
	}
	if (s.RuntimeConfig == nil) != (to.RuntimeConfig == nil) {
		return false
	}
	if s.RuntimeConfig != nil && to.RuntimeConfig != nil {
		if (*s.RuntimeConfig) != (*to.RuntimeConfig) {
			return false
		}
	}

	return true
}

type DestinationField byte

const (
	DestinationFieldID DestinationField = iota + 1
	DestinationFieldWorkspaceID
	DestinationFieldName
	DestinationFieldManagedConnectorID
	DestinationFieldConfig
	DestinationFieldCreatedAt
	DestinationFieldUpdatedAt
	DestinationFieldRuntimeConfig
)

type DestinationFilter struct {
	ID                 filter.Filter[uuid.UUID]
	WorkspaceID        filter.Filter[uuid.UUID]
	Name               filter.Filter[string]
	ManagedConnectorID filter.Filter[uuid.UUID]
	Or                 []*DestinationFilter
	And                []*DestinationFilter
}
type DestinationOrder order.Order[DestinationField]

type Destination struct {
	ID                 uuid.UUID
	WorkspaceID        uuid.UUID
	Name               string
	ManagedConnectorID uuid.UUID
	Config             jsonb
	CreatedAt          time.Time
	UpdatedAt          time.Time
	RuntimeConfig      *jsonb
}

// user code 'Destination methods'
// end user code 'Destination methods'

func (d *Destination) Copy() Destination {
	var result Destination
	result.ID = d.ID
	result.WorkspaceID = d.WorkspaceID
	result.Name = d.Name
	result.ManagedConnectorID = d.ManagedConnectorID
	result.Config = d.Config
	result.CreatedAt = d.CreatedAt
	result.UpdatedAt = d.UpdatedAt
	if d.RuntimeConfig != nil {
		var tmp jsonb
		tmp = (*d.RuntimeConfig)
		result.RuntimeConfig = &tmp
	}

	return result
}
func (d *Destination) Equals(to *Destination) bool {
	if (d == nil) != (to == nil) {
		return false
	}
	if d == nil && to == nil {
		return true
	}
	if d.ID != to.ID {
		return false
	}
	if d.WorkspaceID != to.WorkspaceID {
		return false
	}
	if d.Name != to.Name {
		return false
	}
	if d.ManagedConnectorID != to.ManagedConnectorID {
		return false
	}
	if d.Config != to.Config {
		return false
	}
	if d.CreatedAt != to.CreatedAt {
		return false
	}
	if d.UpdatedAt != to.UpdatedAt {
		return false
	}
	if (d.RuntimeConfig == nil) != (to.RuntimeConfig == nil) {
		return false
	}
	if d.RuntimeConfig != nil && to.RuntimeConfig != nil {
		if (*d.RuntimeConfig) != (*to.RuntimeConfig) {
			return false
		}
	}

	return true
}

type ConnectionField byte

const (
	ConnectionFieldID ConnectionField = iota + 1
	ConnectionFieldWorkspaceID
	ConnectionFieldName
	ConnectionFieldStatus
	ConnectionFieldSourceID
	ConnectionFieldDestinationID
	ConnectionFieldSchedule
	ConnectionFieldSchemaChangePolicy
	ConnectionFieldMaxAttempts
	ConnectionFieldNamespaceDefinition
	ConnectionFieldCustomNamespaceFormat
	ConnectionFieldStreamPrefix
	ConnectionFieldNextScheduledAt
	ConnectionFieldCreatedAt
	ConnectionFieldUpdatedAt
)

type ConnectionFilter struct {
	ID            filter.Filter[uuid.UUID]
	WorkspaceID   filter.Filter[uuid.UUID]
	Name          filter.Filter[string]
	Status        filter.Filter[ConnectionStatus]
	SourceID      filter.Filter[uuid.UUID]
	DestinationID filter.Filter[uuid.UUID]
	Or            []*ConnectionFilter
	And           []*ConnectionFilter
}
type ConnectionOrder order.Order[ConnectionField]

type Connection struct {
	ID                    uuid.UUID
	WorkspaceID           uuid.UUID
	Name                  string
	Status                ConnectionStatus
	SourceID              uuid.UUID
	DestinationID         uuid.UUID
	Schedule              *string
	SchemaChangePolicy    SchemaChangePolicy
	MaxAttempts           int
	NamespaceDefinition   NamespaceDefinition
	CustomNamespaceFormat *string
	StreamPrefix          *string
	NextScheduledAt       *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// user code 'Connection methods'
// end user code 'Connection methods'

func (c *Connection) Copy() Connection {
	var result Connection
	result.ID = c.ID
	result.WorkspaceID = c.WorkspaceID
	result.Name = c.Name
	result.Status = c.Status // enum
	result.SourceID = c.SourceID
	result.DestinationID = c.DestinationID
	if c.Schedule != nil {
		var tmp string
		tmp = (*c.Schedule)
		result.Schedule = &tmp
	}
	result.SchemaChangePolicy = c.SchemaChangePolicy // enum
	result.MaxAttempts = c.MaxAttempts
	result.NamespaceDefinition = c.NamespaceDefinition // enum
	if c.CustomNamespaceFormat != nil {
		var tmp1 string
		tmp1 = (*c.CustomNamespaceFormat)
		result.CustomNamespaceFormat = &tmp1
	}
	if c.StreamPrefix != nil {
		var tmp2 string
		tmp2 = (*c.StreamPrefix)
		result.StreamPrefix = &tmp2
	}
	if c.NextScheduledAt != nil {
		var tmp3 time.Time
		tmp3 = (*c.NextScheduledAt)
		result.NextScheduledAt = &tmp3
	}
	result.CreatedAt = c.CreatedAt
	result.UpdatedAt = c.UpdatedAt

	return result
}
func (c *Connection) Equals(to *Connection) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}
	if c.ID != to.ID {
		return false
	}
	if c.WorkspaceID != to.WorkspaceID {
		return false
	}
	if c.Name != to.Name {
		return false
	}
	if c.Status != to.Status {
		return false
	}
	if c.SourceID != to.SourceID {
		return false
	}
	if c.DestinationID != to.DestinationID {
		return false
	}
	if (c.Schedule == nil) != (to.Schedule == nil) {
		return false
	}
	if c.Schedule != nil && to.Schedule != nil {
		if (*c.Schedule) != (*to.Schedule) {
			return false
		}
	}
	if c.SchemaChangePolicy != to.SchemaChangePolicy {
		return false
	}
	if c.MaxAttempts != to.MaxAttempts {
		return false
	}
	if c.NamespaceDefinition != to.NamespaceDefinition {
		return false
	}
	if (c.CustomNamespaceFormat == nil) != (to.CustomNamespaceFormat == nil) {
		return false
	}
	if c.CustomNamespaceFormat != nil && to.CustomNamespaceFormat != nil {
		if (*c.CustomNamespaceFormat) != (*to.CustomNamespaceFormat) {
			return false
		}
	}
	if (c.StreamPrefix == nil) != (to.StreamPrefix == nil) {
		return false
	}
	if c.StreamPrefix != nil && to.StreamPrefix != nil {
		if (*c.StreamPrefix) != (*to.StreamPrefix) {
			return false
		}
	}
	if (c.NextScheduledAt == nil) != (to.NextScheduledAt == nil) {
		return false
	}
	if c.NextScheduledAt != nil && to.NextScheduledAt != nil {
		if (*c.NextScheduledAt) != (*to.NextScheduledAt) {
			return false
		}
	}
	if c.CreatedAt != to.CreatedAt {
		return false
	}
	if c.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type JobField byte

const (
	JobFieldID JobField = iota + 1
	JobFieldConnectionID
	JobFieldStatus
	JobFieldJobType
	JobFieldScheduledAt
	JobFieldStartedAt
	JobFieldCompletedAt
	JobFieldError
	JobFieldAttempt
	JobFieldMaxAttempts
	JobFieldWorkerID
	JobFieldHeartbeatAt
	JobFieldK8sJobName
	JobFieldFailureReason
	JobFieldCreatedAt
)

type JobFilter struct {
	ID           filter.Filter[uuid.UUID]
	ConnectionID filter.Filter[uuid.UUID]
	Status       filter.Filter[JobStatus]
	JobType      filter.Filter[JobType]
	StartedAt    filter.Filter[*time.Time]
	HeartbeatAt  filter.Filter[*time.Time]
	Or           []*JobFilter
	And          []*JobFilter
}
type JobOrder order.Order[JobField]

type Job struct {
	ID            uuid.UUID
	ConnectionID  uuid.UUID
	Status        JobStatus
	JobType       JobType
	ScheduledAt   time.Time
	StartedAt     *time.Time
	CompletedAt   *time.Time
	Error         *string
	Attempt       int
	MaxAttempts   int
	WorkerID      *string
	HeartbeatAt   *time.Time
	K8sJobName    *string
	FailureReason *string
	CreatedAt     time.Time
}

// user code 'Job methods'
// end user code 'Job methods'

func (j *Job) Copy() Job {
	var result Job
	result.ID = j.ID
	result.ConnectionID = j.ConnectionID
	result.Status = j.Status   // enum
	result.JobType = j.JobType // enum
	result.ScheduledAt = j.ScheduledAt
	if j.StartedAt != nil {
		var tmp time.Time
		tmp = (*j.StartedAt)
		result.StartedAt = &tmp
	}
	if j.CompletedAt != nil {
		var tmp1 time.Time
		tmp1 = (*j.CompletedAt)
		result.CompletedAt = &tmp1
	}
	if j.Error != nil {
		var tmp2 string
		tmp2 = (*j.Error)
		result.Error = &tmp2
	}
	result.Attempt = j.Attempt
	result.MaxAttempts = j.MaxAttempts
	if j.WorkerID != nil {
		var tmp3 string
		tmp3 = (*j.WorkerID)
		result.WorkerID = &tmp3
	}
	if j.HeartbeatAt != nil {
		var tmp4 time.Time
		tmp4 = (*j.HeartbeatAt)
		result.HeartbeatAt = &tmp4
	}
	if j.K8sJobName != nil {
		var tmp5 string
		tmp5 = (*j.K8sJobName)
		result.K8sJobName = &tmp5
	}
	if j.FailureReason != nil {
		var tmp6 string
		tmp6 = (*j.FailureReason)
		result.FailureReason = &tmp6
	}
	result.CreatedAt = j.CreatedAt

	return result
}
func (j *Job) Equals(to *Job) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}
	if j.ID != to.ID {
		return false
	}
	if j.ConnectionID != to.ConnectionID {
		return false
	}
	if j.Status != to.Status {
		return false
	}
	if j.JobType != to.JobType {
		return false
	}
	if j.ScheduledAt != to.ScheduledAt {
		return false
	}
	if (j.StartedAt == nil) != (to.StartedAt == nil) {
		return false
	}
	if j.StartedAt != nil && to.StartedAt != nil {
		if (*j.StartedAt) != (*to.StartedAt) {
			return false
		}
	}
	if (j.CompletedAt == nil) != (to.CompletedAt == nil) {
		return false
	}
	if j.CompletedAt != nil && to.CompletedAt != nil {
		if (*j.CompletedAt) != (*to.CompletedAt) {
			return false
		}
	}
	if (j.Error == nil) != (to.Error == nil) {
		return false
	}
	if j.Error != nil && to.Error != nil {
		if (*j.Error) != (*to.Error) {
			return false
		}
	}
	if j.Attempt != to.Attempt {
		return false
	}
	if j.MaxAttempts != to.MaxAttempts {
		return false
	}
	if (j.WorkerID == nil) != (to.WorkerID == nil) {
		return false
	}
	if j.WorkerID != nil && to.WorkerID != nil {
		if (*j.WorkerID) != (*to.WorkerID) {
			return false
		}
	}
	if (j.HeartbeatAt == nil) != (to.HeartbeatAt == nil) {
		return false
	}
	if j.HeartbeatAt != nil && to.HeartbeatAt != nil {
		if (*j.HeartbeatAt) != (*to.HeartbeatAt) {
			return false
		}
	}
	if (j.K8sJobName == nil) != (to.K8sJobName == nil) {
		return false
	}
	if j.K8sJobName != nil && to.K8sJobName != nil {
		if (*j.K8sJobName) != (*to.K8sJobName) {
			return false
		}
	}
	if (j.FailureReason == nil) != (to.FailureReason == nil) {
		return false
	}
	if j.FailureReason != nil && to.FailureReason != nil {
		if (*j.FailureReason) != (*to.FailureReason) {
			return false
		}
	}
	if j.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type JobAttemptField byte

const (
	JobAttemptFieldID JobAttemptField = iota + 1
	JobAttemptFieldJobID
	JobAttemptFieldAttemptNumber
	JobAttemptFieldStartedAt
	JobAttemptFieldCompletedAt
	JobAttemptFieldError
	JobAttemptFieldSyncStatsJSON
)

type JobAttemptFilter struct {
	ID    filter.Filter[uuid.UUID]
	JobID filter.Filter[uuid.UUID]
	Or    []*JobAttemptFilter
	And   []*JobAttemptFilter
}
type JobAttemptOrder order.Order[JobAttemptField]

type JobAttempt struct {
	ID            uuid.UUID
	JobID         uuid.UUID
	AttemptNumber int
	StartedAt     time.Time
	CompletedAt   *time.Time
	Error         *string
	SyncStatsJSON jsonb
}

// user code 'JobAttempt methods'
// end user code 'JobAttempt methods'

func (j *JobAttempt) Copy() JobAttempt {
	var result JobAttempt
	result.ID = j.ID
	result.JobID = j.JobID
	result.AttemptNumber = j.AttemptNumber
	result.StartedAt = j.StartedAt
	if j.CompletedAt != nil {
		var tmp time.Time
		tmp = (*j.CompletedAt)
		result.CompletedAt = &tmp
	}
	if j.Error != nil {
		var tmp1 string
		tmp1 = (*j.Error)
		result.Error = &tmp1
	}
	result.SyncStatsJSON = j.SyncStatsJSON

	return result
}
func (j *JobAttempt) Equals(to *JobAttempt) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}
	if j.ID != to.ID {
		return false
	}
	if j.JobID != to.JobID {
		return false
	}
	if j.AttemptNumber != to.AttemptNumber {
		return false
	}
	if j.StartedAt != to.StartedAt {
		return false
	}
	if (j.CompletedAt == nil) != (to.CompletedAt == nil) {
		return false
	}
	if j.CompletedAt != nil && to.CompletedAt != nil {
		if (*j.CompletedAt) != (*to.CompletedAt) {
			return false
		}
	}
	if (j.Error == nil) != (to.Error == nil) {
		return false
	}
	if j.Error != nil && to.Error != nil {
		if (*j.Error) != (*to.Error) {
			return false
		}
	}
	if j.SyncStatsJSON != to.SyncStatsJSON {
		return false
	}

	return true
}

type CatalogDiscoveryField byte

const (
	CatalogDiscoveryFieldID CatalogDiscoveryField = iota + 1
	CatalogDiscoveryFieldSourceID
	CatalogDiscoveryFieldVersion
	CatalogDiscoveryFieldCatalogJSON
	CatalogDiscoveryFieldDiscoveredAt
)

type CatalogDiscoveryFilter struct {
	ID       filter.Filter[uuid.UUID]
	SourceID filter.Filter[uuid.UUID]
	Version  filter.Filter[int]
	Or       []*CatalogDiscoveryFilter
	And      []*CatalogDiscoveryFilter
}
type CatalogDiscoveryOrder order.Order[CatalogDiscoveryField]

type CatalogDiscovery struct {
	ID           uuid.UUID
	SourceID     uuid.UUID
	Version      int
	CatalogJSON  jsonb
	DiscoveredAt time.Time
}

// user code 'CatalogDiscovery methods'
// end user code 'CatalogDiscovery methods'

func (c *CatalogDiscovery) Copy() CatalogDiscovery {
	var result CatalogDiscovery
	result.ID = c.ID
	result.SourceID = c.SourceID
	result.Version = c.Version
	result.CatalogJSON = c.CatalogJSON
	result.DiscoveredAt = c.DiscoveredAt

	return result
}
func (c *CatalogDiscovery) Equals(to *CatalogDiscovery) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}
	if c.ID != to.ID {
		return false
	}
	if c.SourceID != to.SourceID {
		return false
	}
	if c.Version != to.Version {
		return false
	}
	if c.CatalogJSON != to.CatalogJSON {
		return false
	}
	if c.DiscoveredAt != to.DiscoveredAt {
		return false
	}

	return true
}

type ConfiguredCatalogField byte

const (
	ConfiguredCatalogFieldID ConfiguredCatalogField = iota + 1
	ConfiguredCatalogFieldConnectionID
	ConfiguredCatalogFieldStreamsJSON
	ConfiguredCatalogFieldCreatedAt
	ConfiguredCatalogFieldUpdatedAt
)

type ConfiguredCatalogFilter struct {
	ID           filter.Filter[uuid.UUID]
	ConnectionID filter.Filter[uuid.UUID]
	Or           []*ConfiguredCatalogFilter
	And          []*ConfiguredCatalogFilter
}
type ConfiguredCatalogOrder order.Order[ConfiguredCatalogField]

type ConfiguredCatalog struct {
	ID           uuid.UUID
	ConnectionID uuid.UUID
	StreamsJSON  jsonb
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// user code 'ConfiguredCatalog methods'
// end user code 'ConfiguredCatalog methods'

func (c *ConfiguredCatalog) Copy() ConfiguredCatalog {
	var result ConfiguredCatalog
	result.ID = c.ID
	result.ConnectionID = c.ConnectionID
	result.StreamsJSON = c.StreamsJSON
	result.CreatedAt = c.CreatedAt
	result.UpdatedAt = c.UpdatedAt

	return result
}
func (c *ConfiguredCatalog) Equals(to *ConfiguredCatalog) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}
	if c.ID != to.ID {
		return false
	}
	if c.ConnectionID != to.ConnectionID {
		return false
	}
	if c.StreamsJSON != to.StreamsJSON {
		return false
	}
	if c.CreatedAt != to.CreatedAt {
		return false
	}
	if c.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type JobLogField byte

const (
	JobLogFieldID JobLogField = iota + 1
	JobLogFieldJobID
	JobLogFieldLogLine
	JobLogFieldCreatedAt
)

type JobLogFilter struct {
	JobID filter.Filter[uuid.UUID]
	Or    []*JobLogFilter
	And   []*JobLogFilter
}
type JobLogOrder order.Order[JobLogField]

type JobLog struct {
	ID        int64
	JobID     uuid.UUID
	LogLine   string
	CreatedAt time.Time
}

// user code 'JobLog methods'
// end user code 'JobLog methods'

func (j *JobLog) Copy() JobLog {
	var result JobLog
	result.ID = j.ID
	result.JobID = j.JobID
	result.LogLine = j.LogLine
	result.CreatedAt = j.CreatedAt

	return result
}
func (j *JobLog) Equals(to *JobLog) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}
	if j.ID != to.ID {
		return false
	}
	if j.JobID != to.JobID {
		return false
	}
	if j.LogLine != to.LogLine {
		return false
	}
	if j.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type ConnectionStateField byte

const (
	ConnectionStateFieldConnectionID ConnectionStateField = iota + 1
	ConnectionStateFieldStateType
	ConnectionStateFieldStateBlob
	ConnectionStateFieldUpdatedAt
)

type ConnectionStateFilter struct {
	ConnectionID filter.Filter[uuid.UUID]
	Or           []*ConnectionStateFilter
	And          []*ConnectionStateFilter
}
type ConnectionStateOrder order.Order[ConnectionStateField]

type ConnectionState struct {
	ConnectionID uuid.UUID
	StateType    string
	StateBlob    jsonb
	UpdatedAt    time.Time
}

// user code 'ConnectionState methods'
// end user code 'ConnectionState methods'

func (c *ConnectionState) Copy() ConnectionState {
	var result ConnectionState
	result.ConnectionID = c.ConnectionID
	result.StateType = c.StateType
	result.StateBlob = c.StateBlob
	result.UpdatedAt = c.UpdatedAt

	return result
}
func (c *ConnectionState) Equals(to *ConnectionState) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}
	if c.ConnectionID != to.ConnectionID {
		return false
	}
	if c.StateType != to.StateType {
		return false
	}
	if c.StateBlob != to.StateBlob {
		return false
	}
	if c.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type CheckPayloadField byte

const (
	CheckPayloadFieldSourceID CheckPayloadField = iota + 1
	CheckPayloadFieldDestinationID
	CheckPayloadFieldManagedConnectorID
	CheckPayloadFieldConfig
)

type CheckPayloadFilter struct {
	Or  []*CheckPayloadFilter
	And []*CheckPayloadFilter
}
type CheckPayloadOrder order.Order[CheckPayloadField]

type CheckPayload struct {
	SourceID           *uuid.UUID
	DestinationID      *uuid.UUID
	ManagedConnectorID uuid.UUID
	Config             *string
}

// user code 'CheckPayload methods'
// end user code 'CheckPayload methods'

func (c *CheckPayload) Copy() CheckPayload {
	var result CheckPayload
	if c.SourceID != nil {
		var tmp uuid.UUID
		tmp = (*c.SourceID)
		result.SourceID = &tmp
	}
	if c.DestinationID != nil {
		var tmp1 uuid.UUID
		tmp1 = (*c.DestinationID)
		result.DestinationID = &tmp1
	}
	result.ManagedConnectorID = c.ManagedConnectorID
	if c.Config != nil {
		var tmp2 string
		tmp2 = (*c.Config)
		result.Config = &tmp2
	}

	return result
}
func (c *CheckPayload) Equals(to *CheckPayload) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}
	if (c.SourceID == nil) != (to.SourceID == nil) {
		return false
	}
	if c.SourceID != nil && to.SourceID != nil {
		if (*c.SourceID) != (*to.SourceID) {
			return false
		}
	}
	if (c.DestinationID == nil) != (to.DestinationID == nil) {
		return false
	}
	if c.DestinationID != nil && to.DestinationID != nil {
		if (*c.DestinationID) != (*to.DestinationID) {
			return false
		}
	}
	if c.ManagedConnectorID != to.ManagedConnectorID {
		return false
	}
	if (c.Config == nil) != (to.Config == nil) {
		return false
	}
	if c.Config != nil && to.Config != nil {
		if (*c.Config) != (*to.Config) {
			return false
		}
	}

	return true
}

type SpecPayloadField byte

const (
	SpecPayloadFieldManagedConnectorID SpecPayloadField = iota + 1
)

type SpecPayloadFilter struct {
	Or  []*SpecPayloadFilter
	And []*SpecPayloadFilter
}
type SpecPayloadOrder order.Order[SpecPayloadField]

type SpecPayload struct {
	ManagedConnectorID uuid.UUID
}

// user code 'SpecPayload methods'
// end user code 'SpecPayload methods'

func (s *SpecPayload) Copy() SpecPayload {
	var result SpecPayload
	result.ManagedConnectorID = s.ManagedConnectorID

	return result
}
func (s *SpecPayload) Equals(to *SpecPayload) bool {
	if (s == nil) != (to == nil) {
		return false
	}
	if s == nil && to == nil {
		return true
	}
	if s.ManagedConnectorID != to.ManagedConnectorID {
		return false
	}

	return true
}

type DiscoverPayloadField byte

const (
	DiscoverPayloadFieldSourceID DiscoverPayloadField = iota + 1
	DiscoverPayloadFieldManagedConnectorID
)

type DiscoverPayloadFilter struct {
	Or  []*DiscoverPayloadFilter
	And []*DiscoverPayloadFilter
}
type DiscoverPayloadOrder order.Order[DiscoverPayloadField]

type DiscoverPayload struct {
	SourceID           uuid.UUID
	ManagedConnectorID uuid.UUID
}

// user code 'DiscoverPayload methods'
// end user code 'DiscoverPayload methods'

func (d *DiscoverPayload) Copy() DiscoverPayload {
	var result DiscoverPayload
	result.SourceID = d.SourceID
	result.ManagedConnectorID = d.ManagedConnectorID

	return result
}
func (d *DiscoverPayload) Equals(to *DiscoverPayload) bool {
	if (d == nil) != (to == nil) {
		return false
	}
	if d == nil && to == nil {
		return true
	}
	if d.SourceID != to.SourceID {
		return false
	}
	if d.ManagedConnectorID != to.ManagedConnectorID {
		return false
	}

	return true
}

type CheckResultField byte

const (
	CheckResultFieldSuccess CheckResultField = iota + 1
	CheckResultFieldMessage
)

type CheckResultFilter struct {
	Or  []*CheckResultFilter
	And []*CheckResultFilter
}
type CheckResultOrder order.Order[CheckResultField]

type CheckResult struct {
	Success bool
	Message string
}

// user code 'CheckResult methods'
// end user code 'CheckResult methods'

func (c *CheckResult) Copy() CheckResult {
	var result CheckResult
	result.Success = c.Success
	result.Message = c.Message

	return result
}
func (c *CheckResult) Equals(to *CheckResult) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}
	if c.Success != to.Success {
		return false
	}
	if c.Message != to.Message {
		return false
	}

	return true
}

type SpecResultField byte

const (
	SpecResultFieldDocumentationURL SpecResultField = iota + 1
	SpecResultFieldChangelogURL
	SpecResultFieldConnectionSpecification
	SpecResultFieldSupportsIncremental
	SpecResultFieldSupportsNormalization
	SpecResultFieldSupportsDBT
	SpecResultFieldSupportedDestinationSyncModes
	SpecResultFieldAdvancedAuth
	SpecResultFieldProtocolVersion
)

type SpecResultFilter struct {
	Or  []*SpecResultFilter
	And []*SpecResultFilter
}
type SpecResultOrder order.Order[SpecResultField]

type SpecResult struct {
	DocumentationURL              string
	ChangelogURL                  string
	ConnectionSpecification       jsonb
	SupportsIncremental           bool
	SupportsNormalization         bool
	SupportsDBT                   bool
	SupportedDestinationSyncModes jsonb
	AdvancedAuth                  jsonb
	ProtocolVersion               string
}

// user code 'SpecResult methods'
// end user code 'SpecResult methods'

func (s *SpecResult) Copy() SpecResult {
	var result SpecResult
	result.DocumentationURL = s.DocumentationURL
	result.ChangelogURL = s.ChangelogURL
	result.ConnectionSpecification = s.ConnectionSpecification
	result.SupportsIncremental = s.SupportsIncremental
	result.SupportsNormalization = s.SupportsNormalization
	result.SupportsDBT = s.SupportsDBT
	result.SupportedDestinationSyncModes = s.SupportedDestinationSyncModes
	result.AdvancedAuth = s.AdvancedAuth
	result.ProtocolVersion = s.ProtocolVersion

	return result
}
func (s *SpecResult) Equals(to *SpecResult) bool {
	if (s == nil) != (to == nil) {
		return false
	}
	if s == nil && to == nil {
		return true
	}
	if s.DocumentationURL != to.DocumentationURL {
		return false
	}
	if s.ChangelogURL != to.ChangelogURL {
		return false
	}
	if s.ConnectionSpecification != to.ConnectionSpecification {
		return false
	}
	if s.SupportsIncremental != to.SupportsIncremental {
		return false
	}
	if s.SupportsNormalization != to.SupportsNormalization {
		return false
	}
	if s.SupportsDBT != to.SupportsDBT {
		return false
	}
	if s.SupportedDestinationSyncModes != to.SupportedDestinationSyncModes {
		return false
	}
	if s.AdvancedAuth != to.AdvancedAuth {
		return false
	}
	if s.ProtocolVersion != to.ProtocolVersion {
		return false
	}

	return true
}

type DiscoverResultField byte

const (
	DiscoverResultFieldCatalog DiscoverResultField = iota + 1
)

type DiscoverResultFilter struct {
	Or  []*DiscoverResultFilter
	And []*DiscoverResultFilter
}
type DiscoverResultOrder order.Order[DiscoverResultField]

type DiscoverResult struct {
	Catalog string
}

// user code 'DiscoverResult methods'
// end user code 'DiscoverResult methods'

func (d *DiscoverResult) Copy() DiscoverResult {
	var result DiscoverResult
	result.Catalog = d.Catalog

	return result
}
func (d *DiscoverResult) Equals(to *DiscoverResult) bool {
	if (d == nil) != (to == nil) {
		return false
	}
	if d == nil && to == nil {
		return true
	}
	if d.Catalog != to.Catalog {
		return false
	}

	return true
}

type WorkspaceSettingsField byte

const (
	WorkspaceSettingsFieldWorkspaceID WorkspaceSettingsField = iota + 1
	WorkspaceSettingsFieldMaxJobsPerWorkspace
	WorkspaceSettingsFieldCreatedAt
	WorkspaceSettingsFieldUpdatedAt
)

type WorkspaceSettingsFilter struct {
	WorkspaceID filter.Filter[uuid.UUID]
	Or          []*WorkspaceSettingsFilter
	And         []*WorkspaceSettingsFilter
}
type WorkspaceSettingsOrder order.Order[WorkspaceSettingsField]

type WorkspaceSettings struct {
	WorkspaceID         uuid.UUID
	MaxJobsPerWorkspace int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// user code 'WorkspaceSettings methods'
// end user code 'WorkspaceSettings methods'

func (w *WorkspaceSettings) Copy() WorkspaceSettings {
	var result WorkspaceSettings
	result.WorkspaceID = w.WorkspaceID
	result.MaxJobsPerWorkspace = w.MaxJobsPerWorkspace
	result.CreatedAt = w.CreatedAt
	result.UpdatedAt = w.UpdatedAt

	return result
}
func (w *WorkspaceSettings) Equals(to *WorkspaceSettings) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.WorkspaceID != to.WorkspaceID {
		return false
	}
	if w.MaxJobsPerWorkspace != to.MaxJobsPerWorkspace {
		return false
	}
	if w.CreatedAt != to.CreatedAt {
		return false
	}
	if w.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type ConnectorTaskField byte

const (
	ConnectorTaskFieldID ConnectorTaskField = iota + 1
	ConnectorTaskFieldWorkspaceID
	ConnectorTaskFieldTaskType
	ConnectorTaskFieldStatus
	ConnectorTaskFieldPayload
	ConnectorTaskFieldResult
	ConnectorTaskFieldErrorMessage
	ConnectorTaskFieldWorkerID
	ConnectorTaskFieldCreatedAt
	ConnectorTaskFieldUpdatedAt
	ConnectorTaskFieldCompletedAt
)

type ConnectorTaskFilter struct {
	ID          filter.Filter[uuid.UUID]
	WorkspaceID filter.Filter[uuid.UUID]
	TaskType    filter.Filter[ConnectorTaskType]
	Status      filter.Filter[ConnectorTaskStatus]
	CreatedAt   filter.Filter[time.Time]
	UpdatedAt   filter.Filter[time.Time]
	CompletedAt filter.Filter[*time.Time]
	Or          []*ConnectorTaskFilter
	And         []*ConnectorTaskFilter
}
type ConnectorTaskOrder order.Order[ConnectorTaskField]

type ConnectorTask struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	TaskType     ConnectorTaskType
	Status       ConnectorTaskStatus
	Payload      ConnectorTaskPayload
	Result       *ConnectorTaskResult
	ErrorMessage *string
	WorkerID     *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  *time.Time
}

// user code 'ConnectorTask methods'
// end user code 'ConnectorTask methods'

func (c *ConnectorTask) Copy() ConnectorTask {
	var result ConnectorTask
	result.ID = c.ID
	result.WorkspaceID = c.WorkspaceID
	result.TaskType = c.TaskType // enum
	result.Status = c.Status     // enum
	result.Payload = copyConnectorTaskPayload(c.Payload)
	if c.Result != nil {
		var tmp ConnectorTaskResult
		tmp = copyConnectorTaskResult((*c.Result))
		result.Result = &tmp
	}
	if c.ErrorMessage != nil {
		var tmp1 string
		tmp1 = (*c.ErrorMessage)
		result.ErrorMessage = &tmp1
	}
	if c.WorkerID != nil {
		var tmp2 string
		tmp2 = (*c.WorkerID)
		result.WorkerID = &tmp2
	}
	result.CreatedAt = c.CreatedAt
	result.UpdatedAt = c.UpdatedAt
	if c.CompletedAt != nil {
		var tmp3 time.Time
		tmp3 = (*c.CompletedAt)
		result.CompletedAt = &tmp3
	}

	return result
}
func (c *ConnectorTask) Equals(to *ConnectorTask) bool {
	if (c == nil) != (to == nil) {
		return false
	}
	if c == nil && to == nil {
		return true
	}
	if c.ID != to.ID {
		return false
	}
	if c.WorkspaceID != to.WorkspaceID {
		return false
	}
	if c.TaskType != to.TaskType {
		return false
	}
	if c.Status != to.Status {
		return false
	}
	if !c.Payload.ConnectorTaskPayloadEquals(to.Payload) {
		return false
	}
	if (c.Result == nil) != (to.Result == nil) {
		return false
	}
	if c.Result != nil && to.Result != nil {
		if !(*c.Result).ConnectorTaskResultEquals((*to.Result)) {
			return false
		}
	}
	if (c.ErrorMessage == nil) != (to.ErrorMessage == nil) {
		return false
	}
	if c.ErrorMessage != nil && to.ErrorMessage != nil {
		if (*c.ErrorMessage) != (*to.ErrorMessage) {
			return false
		}
	}
	if (c.WorkerID == nil) != (to.WorkerID == nil) {
		return false
	}
	if c.WorkerID != nil && to.WorkerID != nil {
		if (*c.WorkerID) != (*to.WorkerID) {
			return false
		}
	}
	if c.CreatedAt != to.CreatedAt {
		return false
	}
	if c.UpdatedAt != to.UpdatedAt {
		return false
	}
	if (c.CompletedAt == nil) != (to.CompletedAt == nil) {
		return false
	}
	if c.CompletedAt != nil && to.CompletedAt != nil {
		if (*c.CompletedAt) != (*to.CompletedAt) {
			return false
		}
	}

	return true
}

type StreamGenerationField byte

const (
	StreamGenerationFieldConnectionID StreamGenerationField = iota + 1
	StreamGenerationFieldStreamNamespace
	StreamGenerationFieldStreamName
	StreamGenerationFieldGenerationID
	StreamGenerationFieldUpdatedAt
)

type StreamGenerationFilter struct {
	ConnectionID    filter.Filter[uuid.UUID]
	StreamNamespace filter.Filter[string]
	StreamName      filter.Filter[string]
	Or              []*StreamGenerationFilter
	And             []*StreamGenerationFilter
}
type StreamGenerationOrder order.Order[StreamGenerationField]

type StreamGeneration struct {
	ConnectionID    uuid.UUID
	StreamNamespace string
	StreamName      string
	GenerationID    int64
	UpdatedAt       time.Time
}

// user code 'StreamGeneration methods'
// end user code 'StreamGeneration methods'

func (s *StreamGeneration) Copy() StreamGeneration {
	var result StreamGeneration
	result.ConnectionID = s.ConnectionID
	result.StreamNamespace = s.StreamNamespace
	result.StreamName = s.StreamName
	result.GenerationID = s.GenerationID
	result.UpdatedAt = s.UpdatedAt

	return result
}
func (s *StreamGeneration) Equals(to *StreamGeneration) bool {
	if (s == nil) != (to == nil) {
		return false
	}
	if s == nil && to == nil {
		return true
	}
	if s.ConnectionID != to.ConnectionID {
		return false
	}
	if s.StreamNamespace != to.StreamNamespace {
		return false
	}
	if s.StreamName != to.StreamName {
		return false
	}
	if s.GenerationID != to.GenerationID {
		return false
	}
	if s.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}
