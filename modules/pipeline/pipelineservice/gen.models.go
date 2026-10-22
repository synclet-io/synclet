package pipelineservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

//sumtype:decl
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

//sumtype:decl
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

//sumtype:decl
type PipelineEventData interface {
	isPipelineEventData()
	PipelineEventDataEquals(PipelineEventData) bool
	// user code 'PipelineEventData methods'
	// end user code 'PipelineEventData methods'
}

func (*JobCreatedEventData) isPipelineEventData() {}
func (j *JobCreatedEventData) PipelineEventDataEquals(to PipelineEventData) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*JobCreatedEventData)
	if !ok {
		return false
	}

	return j.Equals(toTyped)
}
func (*JobUpdatedEventData) isPipelineEventData() {}
func (j *JobUpdatedEventData) PipelineEventDataEquals(to PipelineEventData) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*JobUpdatedEventData)
	if !ok {
		return false
	}

	return j.Equals(toTyped)
}
func (*WorkflowRunStartedEventData) isPipelineEventData() {}
func (w *WorkflowRunStartedEventData) PipelineEventDataEquals(to PipelineEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*WorkflowRunStartedEventData)
	if !ok {
		return false
	}

	return w.Equals(toTyped)
}
func (*WorkflowRunFinishedEventData) isPipelineEventData() {}
func (w *WorkflowRunFinishedEventData) PipelineEventDataEquals(to PipelineEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}

	toTyped, ok := to.(*WorkflowRunFinishedEventData)
	if !ok {
		return false
	}

	return w.Equals(toTyped)
}

func copyPipelineEventData(val PipelineEventData) PipelineEventData {
	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case *JobCreatedEventData:
		valCopy := val.Copy()
		return &valCopy
	case *JobUpdatedEventData:
		valCopy := val.Copy()
		return &valCopy
	case *WorkflowRunStartedEventData:
		valCopy := val.Copy()
		return &valCopy
	case *WorkflowRunFinishedEventData:
		valCopy := val.Copy()
		return &valCopy
	}
	panic("called copyPipelineEventData with invalid type")
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
	ManagedConnectorFieldStarred
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
	Starred       bool
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
	result.Starred = m.Starred

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
	if m.Starred != to.Starred {
		return false
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
	Metadata         RepositoryConnectorMetadata
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
	result.SourceType = r.SourceType    // enum
	result.Metadata = r.Metadata.Copy() // model

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
	if !r.Metadata.Equals(&to.Metadata) {
		return false
	}

	return true
}

type RepositoryConnectorMetadataField byte

const (
	RepositoryConnectorMetadataFieldBreakingChanges RepositoryConnectorMetadataField = iota + 1
	RepositoryConnectorMetadataFieldResourceRequirements
	RepositoryConnectorMetadataFieldMaxSecondsBetweenMessages
	RepositoryConnectorMetadataFieldExternalDocumentationURLs
	RepositoryConnectorMetadataFieldSuggestedStreams
	RepositoryConnectorMetadataFieldAllowedHosts
	RepositoryConnectorMetadataFieldErdURL
	RepositoryConnectorMetadataFieldReleaseDate
	RepositoryConnectorMetadataFieldLanguage
	RepositoryConnectorMetadataFieldTags
	RepositoryConnectorMetadataFieldSupportsRefreshes
	RepositoryConnectorMetadataFieldSupportsFileTransfer
	RepositoryConnectorMetadataFieldSupportsDataActivation
	RepositoryConnectorMetadataFieldMigrationDocumentationURL
)

type RepositoryConnectorMetadataFilter struct {
	Or  []*RepositoryConnectorMetadataFilter
	And []*RepositoryConnectorMetadataFilter
}
type RepositoryConnectorMetadataOrder order.Order[RepositoryConnectorMetadataField]

type RepositoryConnectorMetadata struct {
	BreakingChanges           map[string]BreakingChange
	ResourceRequirements      *ResourceRequirements
	MaxSecondsBetweenMessages *int
	ExternalDocumentationURLs []ExternalDocumentationURL
	SuggestedStreams          []string
	AllowedHosts              []string
	ErdURL                    string
	ReleaseDate               string
	Language                  string
	Tags                      []string
	SupportsRefreshes         bool
	SupportsFileTransfer      bool
	SupportsDataActivation    bool
	MigrationDocumentationURL string
}

// user code 'RepositoryConnectorMetadata methods'
// end user code 'RepositoryConnectorMetadata methods'

func (r *RepositoryConnectorMetadata) Copy() RepositoryConnectorMetadata {
	var result RepositoryConnectorMetadata
	tmp := make(map[string]BreakingChange)
	for k, v := range r.BreakingChanges {
		var keyCopy string
		var valueCopy BreakingChange
		keyCopy = k
		valueCopy = v.Copy() // model
		tmp[keyCopy] = valueCopy
	}
	result.BreakingChanges = tmp
	if r.ResourceRequirements != nil {
		var tmp1 ResourceRequirements
		tmp1 = (*r.ResourceRequirements).Copy() // model
		result.ResourceRequirements = &tmp1
	}
	if r.MaxSecondsBetweenMessages != nil {
		var tmp2 int
		tmp2 = (*r.MaxSecondsBetweenMessages)
		result.MaxSecondsBetweenMessages = &tmp2
	}
	tmp3 := make([]ExternalDocumentationURL, 0, len(r.ExternalDocumentationURLs))
	for _, i := range r.ExternalDocumentationURLs {
		var itemCopy ExternalDocumentationURL
		itemCopy = i.Copy() // model
		tmp3 = append(tmp3, itemCopy)
	}
	result.ExternalDocumentationURLs = tmp3
	tmp4 := make([]string, 0, len(r.SuggestedStreams))
	for _, i1 := range r.SuggestedStreams {
		var itemCopy1 string
		itemCopy1 = i1
		tmp4 = append(tmp4, itemCopy1)
	}
	result.SuggestedStreams = tmp4
	tmp5 := make([]string, 0, len(r.AllowedHosts))
	for _, i2 := range r.AllowedHosts {
		var itemCopy2 string
		itemCopy2 = i2
		tmp5 = append(tmp5, itemCopy2)
	}
	result.AllowedHosts = tmp5
	result.ErdURL = r.ErdURL
	result.ReleaseDate = r.ReleaseDate
	result.Language = r.Language
	tmp6 := make([]string, 0, len(r.Tags))
	for _, i3 := range r.Tags {
		var itemCopy3 string
		itemCopy3 = i3
		tmp6 = append(tmp6, itemCopy3)
	}
	result.Tags = tmp6
	result.SupportsRefreshes = r.SupportsRefreshes
	result.SupportsFileTransfer = r.SupportsFileTransfer
	result.SupportsDataActivation = r.SupportsDataActivation
	result.MigrationDocumentationURL = r.MigrationDocumentationURL

	return result
}
func (r *RepositoryConnectorMetadata) Equals(to *RepositoryConnectorMetadata) bool {
	if (r == nil) != (to == nil) {
		return false
	}
	if r == nil && to == nil {
		return true
	}
	// map comparision
	if len(r.BreakingChanges) != len(to.BreakingChanges) {
		return false
	}
	for k := range r.BreakingChanges {
		valB, ok := to.BreakingChanges[k]
		if !ok {
			return false
		}
		valA := r.BreakingChanges[k]
		if !valA.Equals(&valB) {
			return false
		}
	}
	if (r.ResourceRequirements == nil) != (to.ResourceRequirements == nil) {
		return false
	}
	if r.ResourceRequirements != nil && to.ResourceRequirements != nil {
		if !(*r.ResourceRequirements).Equals(&(*to.ResourceRequirements)) {
			return false
		}
	}
	if (r.MaxSecondsBetweenMessages == nil) != (to.MaxSecondsBetweenMessages == nil) {
		return false
	}
	if r.MaxSecondsBetweenMessages != nil && to.MaxSecondsBetweenMessages != nil {
		if (*r.MaxSecondsBetweenMessages) != (*to.MaxSecondsBetweenMessages) {
			return false
		}
	}
	if len(r.ExternalDocumentationURLs) != len(to.ExternalDocumentationURLs) {
		return false
	}
	for idx := range r.ExternalDocumentationURLs {
		if !r.ExternalDocumentationURLs[idx].Equals(&to.ExternalDocumentationURLs[idx]) {
			return false
		}
	}
	if len(r.SuggestedStreams) != len(to.SuggestedStreams) {
		return false
	}
	for idx1 := range r.SuggestedStreams {
		if r.SuggestedStreams[idx1] != to.SuggestedStreams[idx1] {
			return false
		}
	}
	if len(r.AllowedHosts) != len(to.AllowedHosts) {
		return false
	}
	for idx2 := range r.AllowedHosts {
		if r.AllowedHosts[idx2] != to.AllowedHosts[idx2] {
			return false
		}
	}
	if r.ErdURL != to.ErdURL {
		return false
	}
	if r.ReleaseDate != to.ReleaseDate {
		return false
	}
	if r.Language != to.Language {
		return false
	}
	if len(r.Tags) != len(to.Tags) {
		return false
	}
	for idx3 := range r.Tags {
		if r.Tags[idx3] != to.Tags[idx3] {
			return false
		}
	}
	if r.SupportsRefreshes != to.SupportsRefreshes {
		return false
	}
	if r.SupportsFileTransfer != to.SupportsFileTransfer {
		return false
	}
	if r.SupportsDataActivation != to.SupportsDataActivation {
		return false
	}
	if r.MigrationDocumentationURL != to.MigrationDocumentationURL {
		return false
	}

	return true
}

type BreakingChangeField byte

const (
	BreakingChangeFieldMessage BreakingChangeField = iota + 1
	BreakingChangeFieldMigrationDocumentationURL
	BreakingChangeFieldUpgradeDeadline
)

type BreakingChangeFilter struct {
	Or  []*BreakingChangeFilter
	And []*BreakingChangeFilter
}
type BreakingChangeOrder order.Order[BreakingChangeField]

type BreakingChange struct {
	Message                   string
	MigrationDocumentationURL string
	UpgradeDeadline           string
}

// user code 'BreakingChange methods'
// end user code 'BreakingChange methods'

func (b *BreakingChange) Copy() BreakingChange {
	var result BreakingChange
	result.Message = b.Message
	result.MigrationDocumentationURL = b.MigrationDocumentationURL
	result.UpgradeDeadline = b.UpgradeDeadline

	return result
}
func (b *BreakingChange) Equals(to *BreakingChange) bool {
	if (b == nil) != (to == nil) {
		return false
	}
	if b == nil && to == nil {
		return true
	}
	if b.Message != to.Message {
		return false
	}
	if b.MigrationDocumentationURL != to.MigrationDocumentationURL {
		return false
	}
	if b.UpgradeDeadline != to.UpgradeDeadline {
		return false
	}

	return true
}

type ResourceRequirementsField byte

const (
	ResourceRequirementsFieldJobSpecific ResourceRequirementsField = iota + 1
)

type ResourceRequirementsFilter struct {
	Or  []*ResourceRequirementsFilter
	And []*ResourceRequirementsFilter
}
type ResourceRequirementsOrder order.Order[ResourceRequirementsField]

type ResourceRequirements struct {
	JobSpecific []JobSpecificResourceRequirement
}

// user code 'ResourceRequirements methods'
// end user code 'ResourceRequirements methods'

func (r *ResourceRequirements) Copy() ResourceRequirements {
	var result ResourceRequirements
	tmp := make([]JobSpecificResourceRequirement, 0, len(r.JobSpecific))
	for _, i := range r.JobSpecific {
		var itemCopy JobSpecificResourceRequirement
		itemCopy = i.Copy() // model
		tmp = append(tmp, itemCopy)
	}
	result.JobSpecific = tmp

	return result
}
func (r *ResourceRequirements) Equals(to *ResourceRequirements) bool {
	if (r == nil) != (to == nil) {
		return false
	}
	if r == nil && to == nil {
		return true
	}
	if len(r.JobSpecific) != len(to.JobSpecific) {
		return false
	}
	for idx := range r.JobSpecific {
		if !r.JobSpecific[idx].Equals(&to.JobSpecific[idx]) {
			return false
		}
	}

	return true
}

type JobSpecificResourceRequirementField byte

const (
	JobSpecificResourceRequirementFieldJobType JobSpecificResourceRequirementField = iota + 1
	JobSpecificResourceRequirementFieldResourceRequirements
)

type JobSpecificResourceRequirementFilter struct {
	Or  []*JobSpecificResourceRequirementFilter
	And []*JobSpecificResourceRequirementFilter
}
type JobSpecificResourceRequirementOrder order.Order[JobSpecificResourceRequirementField]

type JobSpecificResourceRequirement struct {
	JobType              string
	ResourceRequirements ResourceRequirementValues
}

// user code 'JobSpecificResourceRequirement methods'
// end user code 'JobSpecificResourceRequirement methods'

func (j *JobSpecificResourceRequirement) Copy() JobSpecificResourceRequirement {
	var result JobSpecificResourceRequirement
	result.JobType = j.JobType
	result.ResourceRequirements = j.ResourceRequirements.Copy() // model

	return result
}
func (j *JobSpecificResourceRequirement) Equals(to *JobSpecificResourceRequirement) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}
	if j.JobType != to.JobType {
		return false
	}
	if !j.ResourceRequirements.Equals(&to.ResourceRequirements) {
		return false
	}

	return true
}

type ResourceRequirementValuesField byte

const (
	ResourceRequirementValuesFieldMemoryLimit ResourceRequirementValuesField = iota + 1
	ResourceRequirementValuesFieldMemoryRequest
	ResourceRequirementValuesFieldCPULimit
	ResourceRequirementValuesFieldCPURequest
)

type ResourceRequirementValuesFilter struct {
	Or  []*ResourceRequirementValuesFilter
	And []*ResourceRequirementValuesFilter
}
type ResourceRequirementValuesOrder order.Order[ResourceRequirementValuesField]

type ResourceRequirementValues struct {
	MemoryLimit   string
	MemoryRequest string
	CPULimit      string
	CPURequest    string
}

// user code 'ResourceRequirementValues methods'
// end user code 'ResourceRequirementValues methods'

func (r *ResourceRequirementValues) Copy() ResourceRequirementValues {
	var result ResourceRequirementValues
	result.MemoryLimit = r.MemoryLimit
	result.MemoryRequest = r.MemoryRequest
	result.CPULimit = r.CPULimit
	result.CPURequest = r.CPURequest

	return result
}
func (r *ResourceRequirementValues) Equals(to *ResourceRequirementValues) bool {
	if (r == nil) != (to == nil) {
		return false
	}
	if r == nil && to == nil {
		return true
	}
	if r.MemoryLimit != to.MemoryLimit {
		return false
	}
	if r.MemoryRequest != to.MemoryRequest {
		return false
	}
	if r.CPULimit != to.CPULimit {
		return false
	}
	if r.CPURequest != to.CPURequest {
		return false
	}

	return true
}

type ExternalDocumentationURLField byte

const (
	ExternalDocumentationURLFieldTitle ExternalDocumentationURLField = iota + 1
	ExternalDocumentationURLFieldType
	ExternalDocumentationURLFieldURL
)

type ExternalDocumentationURLFilter struct {
	Or  []*ExternalDocumentationURLFilter
	And []*ExternalDocumentationURLFilter
}
type ExternalDocumentationURLOrder order.Order[ExternalDocumentationURLField]

type ExternalDocumentationURL struct {
	Title string
	Type  string
	URL   string
}

// user code 'ExternalDocumentationURL methods'
// end user code 'ExternalDocumentationURL methods'

func (e *ExternalDocumentationURL) Copy() ExternalDocumentationURL {
	var result ExternalDocumentationURL
	result.Title = e.Title
	result.Type = e.Type
	result.URL = e.URL

	return result
}
func (e *ExternalDocumentationURL) Equals(to *ExternalDocumentationURL) bool {
	if (e == nil) != (to == nil) {
		return false
	}
	if e == nil && to == nil {
		return true
	}
	if e.Title != to.Title {
		return false
	}
	if e.Type != to.Type {
		return false
	}
	if e.URL != to.URL {
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
	ConnectionFieldStatusReason
	ConnectionFieldMapperSalt
	ConnectionFieldSourceID
	ConnectionFieldDestinationID
	ConnectionFieldSchedule
	ConnectionFieldSchemaChangePolicy
	ConnectionFieldMaxAttempts
	ConnectionFieldTags
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
	Tags          filter.ArrayFilter[string]
	Or            []*ConnectionFilter
	And           []*ConnectionFilter
}
type ConnectionOrder order.Order[ConnectionField]

type Connection struct {
	ID                    uuid.UUID
	WorkspaceID           uuid.UUID
	Name                  string
	Status                ConnectionStatus
	StatusReason          *string
	MapperSalt            string
	SourceID              uuid.UUID
	DestinationID         uuid.UUID
	Schedule              *string
	SchemaChangePolicy    SchemaChangePolicy
	MaxAttempts           int
	Tags                  []string
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
	if c.StatusReason != nil {
		var tmp string
		tmp = (*c.StatusReason)
		result.StatusReason = &tmp
	}
	result.MapperSalt = c.MapperSalt
	result.SourceID = c.SourceID
	result.DestinationID = c.DestinationID
	if c.Schedule != nil {
		var tmp1 string
		tmp1 = (*c.Schedule)
		result.Schedule = &tmp1
	}
	result.SchemaChangePolicy = c.SchemaChangePolicy // enum
	result.MaxAttempts = c.MaxAttempts
	tmp2 := make([]string, 0, len(c.Tags))
	for _, i := range c.Tags {
		var itemCopy string
		itemCopy = i
		tmp2 = append(tmp2, itemCopy)
	}
	result.Tags = tmp2
	result.NamespaceDefinition = c.NamespaceDefinition // enum
	if c.CustomNamespaceFormat != nil {
		var tmp3 string
		tmp3 = (*c.CustomNamespaceFormat)
		result.CustomNamespaceFormat = &tmp3
	}
	if c.StreamPrefix != nil {
		var tmp4 string
		tmp4 = (*c.StreamPrefix)
		result.StreamPrefix = &tmp4
	}
	if c.NextScheduledAt != nil {
		var tmp5 time.Time
		tmp5 = (*c.NextScheduledAt)
		result.NextScheduledAt = &tmp5
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
	if (c.StatusReason == nil) != (to.StatusReason == nil) {
		return false
	}
	if c.StatusReason != nil && to.StatusReason != nil {
		if (*c.StatusReason) != (*to.StatusReason) {
			return false
		}
	}
	if c.MapperSalt != to.MapperSalt {
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
	if len(c.Tags) != len(to.Tags) {
		return false
	}
	for idx := range c.Tags {
		if c.Tags[idx] != to.Tags[idx] {
			return false
		}
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
	JobFieldSyncID
	JobFieldWorkerID
	JobFieldHeartbeatAt
	JobFieldK8sJobName
	JobFieldFailureReason
	JobFieldWorkflowStepRunID
	JobFieldCreatedAt
)

type JobFilter struct {
	ID                filter.Filter[uuid.UUID]
	ConnectionID      filter.Filter[uuid.UUID]
	Status            filter.Filter[JobStatus]
	JobType           filter.Filter[JobType]
	StartedAt         filter.Filter[*time.Time]
	HeartbeatAt       filter.Filter[*time.Time]
	WorkflowStepRunID filter.Filter[*uuid.UUID]
	Or                []*JobFilter
	And               []*JobFilter
}
type JobOrder order.Order[JobField]

type Job struct {
	ID                uuid.UUID
	ConnectionID      uuid.UUID
	Status            JobStatus
	JobType           JobType
	ScheduledAt       time.Time
	StartedAt         *time.Time
	CompletedAt       *time.Time
	Error             *string
	Attempt           int
	MaxAttempts       int
	SyncID            *int64
	WorkerID          *string
	HeartbeatAt       *time.Time
	K8sJobName        *string
	FailureReason     *string
	WorkflowStepRunID *uuid.UUID
	CreatedAt         time.Time
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
	if j.SyncID != nil {
		var tmp3 int64
		tmp3 = (*j.SyncID)
		result.SyncID = &tmp3
	}
	if j.WorkerID != nil {
		var tmp4 string
		tmp4 = (*j.WorkerID)
		result.WorkerID = &tmp4
	}
	if j.HeartbeatAt != nil {
		var tmp5 time.Time
		tmp5 = (*j.HeartbeatAt)
		result.HeartbeatAt = &tmp5
	}
	if j.K8sJobName != nil {
		var tmp6 string
		tmp6 = (*j.K8sJobName)
		result.K8sJobName = &tmp6
	}
	if j.FailureReason != nil {
		var tmp7 string
		tmp7 = (*j.FailureReason)
		result.FailureReason = &tmp7
	}
	if j.WorkflowStepRunID != nil {
		var tmp8 uuid.UUID
		tmp8 = (*j.WorkflowStepRunID)
		result.WorkflowStepRunID = &tmp8
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
	if (j.SyncID == nil) != (to.SyncID == nil) {
		return false
	}
	if j.SyncID != nil && to.SyncID != nil {
		if (*j.SyncID) != (*to.SyncID) {
			return false
		}
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
	if (j.WorkflowStepRunID == nil) != (to.WorkflowStepRunID == nil) {
		return false
	}
	if j.WorkflowStepRunID != nil && to.WorkflowStepRunID != nil {
		if (*j.WorkflowStepRunID) != (*to.WorkflowStepRunID) {
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
	JobLogFieldAttempt
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
	Attempt   int
	LogLine   string
	CreatedAt time.Time
}

// user code 'JobLog methods'
// end user code 'JobLog methods'

func (j *JobLog) Copy() JobLog {
	var result JobLog
	result.ID = j.ID
	result.JobID = j.JobID
	result.Attempt = j.Attempt
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
	if j.Attempt != to.Attempt {
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

type JobLogSegmentField byte

const (
	JobLogSegmentFieldJobID JobLogSegmentField = iota + 1
	JobLogSegmentFieldAttempt
	JobLogSegmentFieldStatus
	JobLogSegmentFieldLineCount
	JobLogSegmentFieldDroppedCount
	JobLogSegmentFieldMarkerLogID
	JobLogSegmentFieldFirstLogID
	JobLogSegmentFieldLastLogID
	JobLogSegmentFieldArchiveKey
	JobLogSegmentFieldSizeBytes
	JobLogSegmentFieldCreatedAt
	JobLogSegmentFieldArchivedAt
)

type JobLogSegmentFilter struct {
	JobID   filter.Filter[uuid.UUID]
	Attempt filter.Filter[int]
	Status  filter.Filter[JobLogSegmentStatus]
	Or      []*JobLogSegmentFilter
	And     []*JobLogSegmentFilter
}
type JobLogSegmentOrder order.Order[JobLogSegmentField]

type JobLogSegment struct {
	JobID        uuid.UUID
	Attempt      int
	Status       JobLogSegmentStatus
	LineCount    int
	DroppedCount int
	MarkerLogID  *int64
	FirstLogID   *int64
	LastLogID    *int64
	ArchiveKey   *string
	SizeBytes    int64
	CreatedAt    time.Time
	ArchivedAt   *time.Time
}

// user code 'JobLogSegment methods'
// end user code 'JobLogSegment methods'

func (j *JobLogSegment) Copy() JobLogSegment {
	var result JobLogSegment
	result.JobID = j.JobID
	result.Attempt = j.Attempt
	result.Status = j.Status // enum
	result.LineCount = j.LineCount
	result.DroppedCount = j.DroppedCount
	if j.MarkerLogID != nil {
		var tmp int64
		tmp = (*j.MarkerLogID)
		result.MarkerLogID = &tmp
	}
	if j.FirstLogID != nil {
		var tmp1 int64
		tmp1 = (*j.FirstLogID)
		result.FirstLogID = &tmp1
	}
	if j.LastLogID != nil {
		var tmp2 int64
		tmp2 = (*j.LastLogID)
		result.LastLogID = &tmp2
	}
	if j.ArchiveKey != nil {
		var tmp3 string
		tmp3 = (*j.ArchiveKey)
		result.ArchiveKey = &tmp3
	}
	result.SizeBytes = j.SizeBytes
	result.CreatedAt = j.CreatedAt
	if j.ArchivedAt != nil {
		var tmp4 time.Time
		tmp4 = (*j.ArchivedAt)
		result.ArchivedAt = &tmp4
	}

	return result
}
func (j *JobLogSegment) Equals(to *JobLogSegment) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}
	if j.JobID != to.JobID {
		return false
	}
	if j.Attempt != to.Attempt {
		return false
	}
	if j.Status != to.Status {
		return false
	}
	if j.LineCount != to.LineCount {
		return false
	}
	if j.DroppedCount != to.DroppedCount {
		return false
	}
	if (j.MarkerLogID == nil) != (to.MarkerLogID == nil) {
		return false
	}
	if j.MarkerLogID != nil && to.MarkerLogID != nil {
		if (*j.MarkerLogID) != (*to.MarkerLogID) {
			return false
		}
	}
	if (j.FirstLogID == nil) != (to.FirstLogID == nil) {
		return false
	}
	if j.FirstLogID != nil && to.FirstLogID != nil {
		if (*j.FirstLogID) != (*to.FirstLogID) {
			return false
		}
	}
	if (j.LastLogID == nil) != (to.LastLogID == nil) {
		return false
	}
	if j.LastLogID != nil && to.LastLogID != nil {
		if (*j.LastLogID) != (*to.LastLogID) {
			return false
		}
	}
	if (j.ArchiveKey == nil) != (to.ArchiveKey == nil) {
		return false
	}
	if j.ArchiveKey != nil && to.ArchiveKey != nil {
		if (*j.ArchiveKey) != (*to.ArchiveKey) {
			return false
		}
	}
	if j.SizeBytes != to.SizeBytes {
		return false
	}
	if j.CreatedAt != to.CreatedAt {
		return false
	}
	if (j.ArchivedAt == nil) != (to.ArchivedAt == nil) {
		return false
	}
	if j.ArchivedAt != nil && to.ArchivedAt != nil {
		if (*j.ArchivedAt) != (*to.ArchivedAt) {
			return false
		}
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

type ExecutorField byte

const (
	ExecutorFieldID ExecutorField = iota + 1
	ExecutorFieldInstanceID
	ExecutorFieldType
	ExecutorFieldVersion
	ExecutorFieldCapacity
	ExecutorFieldCapabilities
	ExecutorFieldResourceDefaults
	ExecutorFieldFirstSeenAt
	ExecutorFieldLastSeenAt
)

type ExecutorFilter struct {
	ID         filter.Filter[uuid.UUID]
	InstanceID filter.Filter[string]
	Type       filter.Filter[ExecutorType]
	Or         []*ExecutorFilter
	And        []*ExecutorFilter
}
type ExecutorOrder order.Order[ExecutorField]

type Executor struct {
	ID               uuid.UUID
	InstanceID       string
	Type             ExecutorType
	Version          string
	Capacity         int
	Capabilities     ExecutorCapabilities
	ResourceDefaults ExecutorResourceDefaults
	FirstSeenAt      time.Time
	LastSeenAt       time.Time
}

// user code 'Executor methods'
// end user code 'Executor methods'

func (e *Executor) Copy() Executor {
	var result Executor
	result.ID = e.ID
	result.InstanceID = e.InstanceID
	result.Type = e.Type // enum
	result.Version = e.Version
	result.Capacity = e.Capacity
	result.Capabilities = e.Capabilities.Copy()         // model
	result.ResourceDefaults = e.ResourceDefaults.Copy() // model
	result.FirstSeenAt = e.FirstSeenAt
	result.LastSeenAt = e.LastSeenAt

	return result
}
func (e *Executor) Equals(to *Executor) bool {
	if (e == nil) != (to == nil) {
		return false
	}
	if e == nil && to == nil {
		return true
	}
	if e.ID != to.ID {
		return false
	}
	if e.InstanceID != to.InstanceID {
		return false
	}
	if e.Type != to.Type {
		return false
	}
	if e.Version != to.Version {
		return false
	}
	if e.Capacity != to.Capacity {
		return false
	}
	if !e.Capabilities.Equals(&to.Capabilities) {
		return false
	}
	if !e.ResourceDefaults.Equals(&to.ResourceDefaults) {
		return false
	}
	if e.FirstSeenAt != to.FirstSeenAt {
		return false
	}
	if e.LastSeenAt != to.LastSeenAt {
		return false
	}

	return true
}

type ExecutorCapabilitiesField byte

const (
	ExecutorCapabilitiesFieldNativeDestinations ExecutorCapabilitiesField = iota + 1
)

type ExecutorCapabilitiesFilter struct {
	Or  []*ExecutorCapabilitiesFilter
	And []*ExecutorCapabilitiesFilter
}
type ExecutorCapabilitiesOrder order.Order[ExecutorCapabilitiesField]

type ExecutorCapabilities struct {
	NativeDestinations []string
}

// user code 'ExecutorCapabilities methods'
// end user code 'ExecutorCapabilities methods'

func (e *ExecutorCapabilities) Copy() ExecutorCapabilities {
	var result ExecutorCapabilities
	tmp := make([]string, 0, len(e.NativeDestinations))
	for _, i := range e.NativeDestinations {
		var itemCopy string
		itemCopy = i
		tmp = append(tmp, itemCopy)
	}
	result.NativeDestinations = tmp

	return result
}
func (e *ExecutorCapabilities) Equals(to *ExecutorCapabilities) bool {
	if (e == nil) != (to == nil) {
		return false
	}
	if e == nil && to == nil {
		return true
	}
	if len(e.NativeDestinations) != len(to.NativeDestinations) {
		return false
	}
	for idx := range e.NativeDestinations {
		if e.NativeDestinations[idx] != to.NativeDestinations[idx] {
			return false
		}
	}

	return true
}

type ExecutorResourceDefaultsField byte

const (
	ExecutorResourceDefaultsFieldCPURequest ExecutorResourceDefaultsField = iota + 1
	ExecutorResourceDefaultsFieldCPULimit
	ExecutorResourceDefaultsFieldMemoryRequest
	ExecutorResourceDefaultsFieldMemoryLimit
	ExecutorResourceDefaultsFieldCPUExecutorDefault
	ExecutorResourceDefaultsFieldMemoryExecutorDefault
)

type ExecutorResourceDefaultsFilter struct {
	Or  []*ExecutorResourceDefaultsFilter
	And []*ExecutorResourceDefaultsFilter
}
type ExecutorResourceDefaultsOrder order.Order[ExecutorResourceDefaultsField]

type ExecutorResourceDefaults struct {
	CPURequest            string
	CPULimit              string
	MemoryRequest         string
	MemoryLimit           string
	CPUExecutorDefault    bool
	MemoryExecutorDefault bool
}

// user code 'ExecutorResourceDefaults methods'
// end user code 'ExecutorResourceDefaults methods'

func (e *ExecutorResourceDefaults) Copy() ExecutorResourceDefaults {
	var result ExecutorResourceDefaults
	result.CPURequest = e.CPURequest
	result.CPULimit = e.CPULimit
	result.MemoryRequest = e.MemoryRequest
	result.MemoryLimit = e.MemoryLimit
	result.CPUExecutorDefault = e.CPUExecutorDefault
	result.MemoryExecutorDefault = e.MemoryExecutorDefault

	return result
}
func (e *ExecutorResourceDefaults) Equals(to *ExecutorResourceDefaults) bool {
	if (e == nil) != (to == nil) {
		return false
	}
	if e == nil && to == nil {
		return true
	}
	if e.CPURequest != to.CPURequest {
		return false
	}
	if e.CPULimit != to.CPULimit {
		return false
	}
	if e.MemoryRequest != to.MemoryRequest {
		return false
	}
	if e.MemoryLimit != to.MemoryLimit {
		return false
	}
	if e.CPUExecutorDefault != to.CPUExecutorDefault {
		return false
	}
	if e.MemoryExecutorDefault != to.MemoryExecutorDefault {
		return false
	}

	return true
}

type WorkflowField byte

const (
	WorkflowFieldID WorkflowField = iota + 1
	WorkflowFieldWorkspaceID
	WorkflowFieldName
	WorkflowFieldSchedule
	WorkflowFieldNextScheduledAt
	WorkflowFieldCreatedAt
	WorkflowFieldUpdatedAt
)

type WorkflowFilter struct {
	ID              filter.Filter[uuid.UUID]
	WorkspaceID     filter.Filter[uuid.UUID]
	Name            filter.Filter[string]
	NextScheduledAt filter.Filter[*time.Time]
	Or              []*WorkflowFilter
	And             []*WorkflowFilter
}
type WorkflowOrder order.Order[WorkflowField]

type Workflow struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	Name            string
	Schedule        *string
	NextScheduledAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// user code 'Workflow methods'
// end user code 'Workflow methods'

func (w *Workflow) Copy() Workflow {
	var result Workflow
	result.ID = w.ID
	result.WorkspaceID = w.WorkspaceID
	result.Name = w.Name
	if w.Schedule != nil {
		var tmp string
		tmp = (*w.Schedule)
		result.Schedule = &tmp
	}
	if w.NextScheduledAt != nil {
		var tmp1 time.Time
		tmp1 = (*w.NextScheduledAt)
		result.NextScheduledAt = &tmp1
	}
	result.CreatedAt = w.CreatedAt
	result.UpdatedAt = w.UpdatedAt

	return result
}
func (w *Workflow) Equals(to *Workflow) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.WorkspaceID != to.WorkspaceID {
		return false
	}
	if w.Name != to.Name {
		return false
	}
	if (w.Schedule == nil) != (to.Schedule == nil) {
		return false
	}
	if w.Schedule != nil && to.Schedule != nil {
		if (*w.Schedule) != (*to.Schedule) {
			return false
		}
	}
	if (w.NextScheduledAt == nil) != (to.NextScheduledAt == nil) {
		return false
	}
	if w.NextScheduledAt != nil && to.NextScheduledAt != nil {
		if (*w.NextScheduledAt) != (*to.NextScheduledAt) {
			return false
		}
	}
	if w.CreatedAt != to.CreatedAt {
		return false
	}
	if w.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type WorkflowStepField byte

const (
	WorkflowStepFieldID WorkflowStepField = iota + 1
	WorkflowStepFieldWorkflowID
	WorkflowStepFieldConnectionID
)

type WorkflowStepFilter struct {
	ID           filter.Filter[uuid.UUID]
	WorkflowID   filter.Filter[uuid.UUID]
	ConnectionID filter.Filter[uuid.UUID]
	Or           []*WorkflowStepFilter
	And          []*WorkflowStepFilter
}
type WorkflowStepOrder order.Order[WorkflowStepField]

type WorkflowStep struct {
	ID           uuid.UUID
	WorkflowID   uuid.UUID
	ConnectionID uuid.UUID
}

// user code 'WorkflowStep methods'
// end user code 'WorkflowStep methods'

func (w *WorkflowStep) Copy() WorkflowStep {
	var result WorkflowStep
	result.ID = w.ID
	result.WorkflowID = w.WorkflowID
	result.ConnectionID = w.ConnectionID

	return result
}
func (w *WorkflowStep) Equals(to *WorkflowStep) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.WorkflowID != to.WorkflowID {
		return false
	}
	if w.ConnectionID != to.ConnectionID {
		return false
	}

	return true
}

type WorkflowStepDependencyField byte

const (
	WorkflowStepDependencyFieldID WorkflowStepDependencyField = iota + 1
	WorkflowStepDependencyFieldWorkflowID
	WorkflowStepDependencyFieldStepID
	WorkflowStepDependencyFieldUpstreamStepID
)

type WorkflowStepDependencyFilter struct {
	ID             filter.Filter[uuid.UUID]
	WorkflowID     filter.Filter[uuid.UUID]
	StepID         filter.Filter[uuid.UUID]
	UpstreamStepID filter.Filter[uuid.UUID]
	Or             []*WorkflowStepDependencyFilter
	And            []*WorkflowStepDependencyFilter
}
type WorkflowStepDependencyOrder order.Order[WorkflowStepDependencyField]

type WorkflowStepDependency struct {
	ID             uuid.UUID
	WorkflowID     uuid.UUID
	StepID         uuid.UUID
	UpstreamStepID uuid.UUID
}

// user code 'WorkflowStepDependency methods'
// end user code 'WorkflowStepDependency methods'

func (w *WorkflowStepDependency) Copy() WorkflowStepDependency {
	var result WorkflowStepDependency
	result.ID = w.ID
	result.WorkflowID = w.WorkflowID
	result.StepID = w.StepID
	result.UpstreamStepID = w.UpstreamStepID

	return result
}
func (w *WorkflowStepDependency) Equals(to *WorkflowStepDependency) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.WorkflowID != to.WorkflowID {
		return false
	}
	if w.StepID != to.StepID {
		return false
	}
	if w.UpstreamStepID != to.UpstreamStepID {
		return false
	}

	return true
}

type WorkflowRunField byte

const (
	WorkflowRunFieldID WorkflowRunField = iota + 1
	WorkflowRunFieldWorkflowID
	WorkflowRunFieldStatus
	WorkflowRunFieldTrigger
	WorkflowRunFieldStartedAt
	WorkflowRunFieldCompletedAt
	WorkflowRunFieldCreatedAt
)

type WorkflowRunFilter struct {
	ID         filter.Filter[uuid.UUID]
	WorkflowID filter.Filter[uuid.UUID]
	Status     filter.Filter[WorkflowRunStatus]
	Trigger    filter.Filter[WorkflowRunTrigger]
	Or         []*WorkflowRunFilter
	And        []*WorkflowRunFilter
}
type WorkflowRunOrder order.Order[WorkflowRunField]

type WorkflowRun struct {
	ID          uuid.UUID
	WorkflowID  uuid.UUID
	Status      WorkflowRunStatus
	Trigger     WorkflowRunTrigger
	StartedAt   time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
}

// user code 'WorkflowRun methods'
// end user code 'WorkflowRun methods'

func (w *WorkflowRun) Copy() WorkflowRun {
	var result WorkflowRun
	result.ID = w.ID
	result.WorkflowID = w.WorkflowID
	result.Status = w.Status   // enum
	result.Trigger = w.Trigger // enum
	result.StartedAt = w.StartedAt
	if w.CompletedAt != nil {
		var tmp time.Time
		tmp = (*w.CompletedAt)
		result.CompletedAt = &tmp
	}
	result.CreatedAt = w.CreatedAt

	return result
}
func (w *WorkflowRun) Equals(to *WorkflowRun) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.WorkflowID != to.WorkflowID {
		return false
	}
	if w.Status != to.Status {
		return false
	}
	if w.Trigger != to.Trigger {
		return false
	}
	if w.StartedAt != to.StartedAt {
		return false
	}
	if (w.CompletedAt == nil) != (to.CompletedAt == nil) {
		return false
	}
	if w.CompletedAt != nil && to.CompletedAt != nil {
		if (*w.CompletedAt) != (*to.CompletedAt) {
			return false
		}
	}
	if w.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type WorkflowStepRunField byte

const (
	WorkflowStepRunFieldID WorkflowStepRunField = iota + 1
	WorkflowStepRunFieldRunID
	WorkflowStepRunFieldStepID
	WorkflowStepRunFieldConnectionID
	WorkflowStepRunFieldStatus
	WorkflowStepRunFieldJobID
	WorkflowStepRunFieldError
	WorkflowStepRunFieldStartedAt
	WorkflowStepRunFieldCompletedAt
)

type WorkflowStepRunFilter struct {
	ID           filter.Filter[uuid.UUID]
	RunID        filter.Filter[uuid.UUID]
	StepID       filter.Filter[uuid.UUID]
	ConnectionID filter.Filter[uuid.UUID]
	Status       filter.Filter[WorkflowStepRunStatus]
	JobID        filter.Filter[*uuid.UUID]
	Or           []*WorkflowStepRunFilter
	And          []*WorkflowStepRunFilter
}
type WorkflowStepRunOrder order.Order[WorkflowStepRunField]

type WorkflowStepRun struct {
	ID           uuid.UUID
	RunID        uuid.UUID
	StepID       uuid.UUID
	ConnectionID uuid.UUID
	Status       WorkflowStepRunStatus
	JobID        *uuid.UUID
	Error        *string
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// user code 'WorkflowStepRun methods'
// end user code 'WorkflowStepRun methods'

func (w *WorkflowStepRun) Copy() WorkflowStepRun {
	var result WorkflowStepRun
	result.ID = w.ID
	result.RunID = w.RunID
	result.StepID = w.StepID
	result.ConnectionID = w.ConnectionID
	result.Status = w.Status // enum
	if w.JobID != nil {
		var tmp uuid.UUID
		tmp = (*w.JobID)
		result.JobID = &tmp
	}
	if w.Error != nil {
		var tmp1 string
		tmp1 = (*w.Error)
		result.Error = &tmp1
	}
	if w.StartedAt != nil {
		var tmp2 time.Time
		tmp2 = (*w.StartedAt)
		result.StartedAt = &tmp2
	}
	if w.CompletedAt != nil {
		var tmp3 time.Time
		tmp3 = (*w.CompletedAt)
		result.CompletedAt = &tmp3
	}

	return result
}
func (w *WorkflowStepRun) Equals(to *WorkflowStepRun) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.RunID != to.RunID {
		return false
	}
	if w.StepID != to.StepID {
		return false
	}
	if w.ConnectionID != to.ConnectionID {
		return false
	}
	if w.Status != to.Status {
		return false
	}
	if (w.JobID == nil) != (to.JobID == nil) {
		return false
	}
	if w.JobID != nil && to.JobID != nil {
		if (*w.JobID) != (*to.JobID) {
			return false
		}
	}
	if (w.Error == nil) != (to.Error == nil) {
		return false
	}
	if w.Error != nil && to.Error != nil {
		if (*w.Error) != (*to.Error) {
			return false
		}
	}
	if (w.StartedAt == nil) != (to.StartedAt == nil) {
		return false
	}
	if w.StartedAt != nil && to.StartedAt != nil {
		if (*w.StartedAt) != (*to.StartedAt) {
			return false
		}
	}
	if (w.CompletedAt == nil) != (to.CompletedAt == nil) {
		return false
	}
	if w.CompletedAt != nil && to.CompletedAt != nil {
		if (*w.CompletedAt) != (*to.CompletedAt) {
			return false
		}
	}

	return true
}

type PipelineEventField byte

const (
	PipelineEventFieldEventType PipelineEventField = iota + 1
	PipelineEventFieldData
)

type PipelineEventFilter struct {
	Or  []*PipelineEventFilter
	And []*PipelineEventFilter
}
type PipelineEventOrder order.Order[PipelineEventField]

type PipelineEvent struct {
	EventType PipelineEventType
	Data      PipelineEventData
}

// user code 'PipelineEvent methods'
// end user code 'PipelineEvent methods'

func (p *PipelineEvent) Copy() PipelineEvent {
	var result PipelineEvent
	result.EventType = p.EventType // enum
	result.Data = copyPipelineEventData(p.Data)

	return result
}
func (p *PipelineEvent) Equals(to *PipelineEvent) bool {
	if (p == nil) != (to == nil) {
		return false
	}
	if p == nil && to == nil {
		return true
	}
	if p.EventType != to.EventType {
		return false
	}
	if !p.Data.PipelineEventDataEquals(to.Data) {
		return false
	}

	return true
}

type JobCreatedEventDataField byte

const (
	JobCreatedEventDataFieldWorkspaceID JobCreatedEventDataField = iota + 1
	JobCreatedEventDataFieldJob
)

type JobCreatedEventDataFilter struct {
	Or  []*JobCreatedEventDataFilter
	And []*JobCreatedEventDataFilter
}
type JobCreatedEventDataOrder order.Order[JobCreatedEventDataField]

type JobCreatedEventData struct {
	WorkspaceID uuid.UUID
	Job         *Job
}

// user code 'JobCreatedEventData methods'
// end user code 'JobCreatedEventData methods'

func (j *JobCreatedEventData) Copy() JobCreatedEventData {
	var result JobCreatedEventData
	result.WorkspaceID = j.WorkspaceID
	if j.Job != nil {
		var tmp Job
		tmp = (*j.Job).Copy() // model
		result.Job = &tmp
	}

	return result
}
func (j *JobCreatedEventData) Equals(to *JobCreatedEventData) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}
	if j.WorkspaceID != to.WorkspaceID {
		return false
	}
	if (j.Job == nil) != (to.Job == nil) {
		return false
	}
	if j.Job != nil && to.Job != nil {
		if !(*j.Job).Equals(&(*to.Job)) {
			return false
		}
	}

	return true
}

type JobUpdatedEventDataField byte

const (
	JobUpdatedEventDataFieldWorkspaceID JobUpdatedEventDataField = iota + 1
	JobUpdatedEventDataFieldOldData
	JobUpdatedEventDataFieldNewData
)

type JobUpdatedEventDataFilter struct {
	Or  []*JobUpdatedEventDataFilter
	And []*JobUpdatedEventDataFilter
}
type JobUpdatedEventDataOrder order.Order[JobUpdatedEventDataField]

type JobUpdatedEventData struct {
	WorkspaceID uuid.UUID
	OldData     *Job
	NewData     *Job
}

// user code 'JobUpdatedEventData methods'
// end user code 'JobUpdatedEventData methods'

func (j *JobUpdatedEventData) Copy() JobUpdatedEventData {
	var result JobUpdatedEventData
	result.WorkspaceID = j.WorkspaceID
	if j.OldData != nil {
		var tmp Job
		tmp = (*j.OldData).Copy() // model
		result.OldData = &tmp
	}
	if j.NewData != nil {
		var tmp1 Job
		tmp1 = (*j.NewData).Copy() // model
		result.NewData = &tmp1
	}

	return result
}
func (j *JobUpdatedEventData) Equals(to *JobUpdatedEventData) bool {
	if (j == nil) != (to == nil) {
		return false
	}
	if j == nil && to == nil {
		return true
	}
	if j.WorkspaceID != to.WorkspaceID {
		return false
	}
	if (j.OldData == nil) != (to.OldData == nil) {
		return false
	}
	if j.OldData != nil && to.OldData != nil {
		if !(*j.OldData).Equals(&(*to.OldData)) {
			return false
		}
	}
	if (j.NewData == nil) != (to.NewData == nil) {
		return false
	}
	if j.NewData != nil && to.NewData != nil {
		if !(*j.NewData).Equals(&(*to.NewData)) {
			return false
		}
	}

	return true
}

type WorkflowRunStartedEventDataField byte

const (
	WorkflowRunStartedEventDataFieldWorkspaceID WorkflowRunStartedEventDataField = iota + 1
	WorkflowRunStartedEventDataFieldWorkflowID
	WorkflowRunStartedEventDataFieldRunID
)

type WorkflowRunStartedEventDataFilter struct {
	Or  []*WorkflowRunStartedEventDataFilter
	And []*WorkflowRunStartedEventDataFilter
}
type WorkflowRunStartedEventDataOrder order.Order[WorkflowRunStartedEventDataField]

type WorkflowRunStartedEventData struct {
	WorkspaceID uuid.UUID
	WorkflowID  uuid.UUID
	RunID       uuid.UUID
}

// user code 'WorkflowRunStartedEventData methods'
// end user code 'WorkflowRunStartedEventData methods'

func (w *WorkflowRunStartedEventData) Copy() WorkflowRunStartedEventData {
	var result WorkflowRunStartedEventData
	result.WorkspaceID = w.WorkspaceID
	result.WorkflowID = w.WorkflowID
	result.RunID = w.RunID

	return result
}
func (w *WorkflowRunStartedEventData) Equals(to *WorkflowRunStartedEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.WorkspaceID != to.WorkspaceID {
		return false
	}
	if w.WorkflowID != to.WorkflowID {
		return false
	}
	if w.RunID != to.RunID {
		return false
	}

	return true
}

type WorkflowRunFinishedEventDataField byte

const (
	WorkflowRunFinishedEventDataFieldWorkspaceID WorkflowRunFinishedEventDataField = iota + 1
	WorkflowRunFinishedEventDataFieldWorkflowID
	WorkflowRunFinishedEventDataFieldWorkflowName
	WorkflowRunFinishedEventDataFieldRunID
	WorkflowRunFinishedEventDataFieldStatus
	WorkflowRunFinishedEventDataFieldError
	WorkflowRunFinishedEventDataFieldConnectionIDs
)

type WorkflowRunFinishedEventDataFilter struct {
	Or  []*WorkflowRunFinishedEventDataFilter
	And []*WorkflowRunFinishedEventDataFilter
}
type WorkflowRunFinishedEventDataOrder order.Order[WorkflowRunFinishedEventDataField]

type WorkflowRunFinishedEventData struct {
	WorkspaceID   uuid.UUID
	WorkflowID    uuid.UUID
	WorkflowName  string
	RunID         uuid.UUID
	Status        WorkflowRunStatus
	Error         *string
	ConnectionIDs []uuid.UUID
}

// user code 'WorkflowRunFinishedEventData methods'
// end user code 'WorkflowRunFinishedEventData methods'

func (w *WorkflowRunFinishedEventData) Copy() WorkflowRunFinishedEventData {
	var result WorkflowRunFinishedEventData
	result.WorkspaceID = w.WorkspaceID
	result.WorkflowID = w.WorkflowID
	result.WorkflowName = w.WorkflowName
	result.RunID = w.RunID
	result.Status = w.Status // enum
	if w.Error != nil {
		var tmp string
		tmp = (*w.Error)
		result.Error = &tmp
	}
	tmp1 := make([]uuid.UUID, 0, len(w.ConnectionIDs))
	for _, i := range w.ConnectionIDs {
		var itemCopy uuid.UUID
		itemCopy = i
		tmp1 = append(tmp1, itemCopy)
	}
	result.ConnectionIDs = tmp1

	return result
}
func (w *WorkflowRunFinishedEventData) Equals(to *WorkflowRunFinishedEventData) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.WorkspaceID != to.WorkspaceID {
		return false
	}
	if w.WorkflowID != to.WorkflowID {
		return false
	}
	if w.WorkflowName != to.WorkflowName {
		return false
	}
	if w.RunID != to.RunID {
		return false
	}
	if w.Status != to.Status {
		return false
	}
	if (w.Error == nil) != (to.Error == nil) {
		return false
	}
	if w.Error != nil && to.Error != nil {
		if (*w.Error) != (*to.Error) {
			return false
		}
	}
	if len(w.ConnectionIDs) != len(to.ConnectionIDs) {
		return false
	}
	for idx := range w.ConnectionIDs {
		if w.ConnectionIDs[idx] != to.ConnectionIDs[idx] {
			return false
		}
	}

	return true
}
