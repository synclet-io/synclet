package pipelinestorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	fmt "fmt"
	time "time"

	uuid "github.com/google/uuid"
	errors "github.com/pkg/errors"

	pipelineservice "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonManagedConnector struct {
	ID            uuid.UUID  `json:"id"`
	WorkspaceID   uuid.UUID  `json:"workspace_id"`
	DockerImage   string     `json:"docker_image"`
	DockerTag     string     `json:"docker_tag"`
	Name          string     `json:"name"`
	ConnectorType string     `json:"connector_type"`
	Spec          jsonb      `json:"spec"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	RepositoryID  *uuid.UUID `json:"repository_id"`
}

func (m *jsonManagedConnector) Scan(value any) error {
	return json.Unmarshal(value.([]byte), m)
}

func (m jsonManagedConnector) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func convertManagedConnectorToJsonModel(src *pipelineservice.ManagedConnector) (*jsonManagedConnector, error) {
	result := &jsonManagedConnector{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.DockerImage = src.DockerImage
	result.DockerTag = src.DockerTag
	result.Name = src.Name
	tmp5, err := convertConnectorTypeToDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp5
	result.Spec = src.Spec
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.RepositoryID == nil {
		result.RepositoryID = nil
	} else {
		result.RepositoryID = toPtr(fromPtr(src.RepositoryID))
	}
	return result, nil
}

func convertManagedConnectorFromJsonModel(src *jsonManagedConnector) (*pipelineservice.ManagedConnector, error) {
	result := &pipelineservice.ManagedConnector{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.DockerImage = src.DockerImage
	result.DockerTag = src.DockerTag
	result.Name = src.Name
	tmp16, err := convertConnectorTypeFromDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp16
	result.Spec = src.Spec
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.RepositoryID == nil {
		result.RepositoryID = nil
	} else {
		result.RepositoryID = toPtr(fromPtr(src.RepositoryID))
	}
	return result, nil
}

type jsonRepository struct {
	ID             uuid.UUID  `json:"id"`
	WorkspaceID    uuid.UUID  `json:"workspace_id"`
	Name           string     `json:"name"`
	URL            string     `json:"url"`
	AuthHeader     *string    `json:"auth_header"`
	Status         string     `json:"status"`
	LastSyncedAt   *time.Time `json:"last_synced_at"`
	ConnectorCount int        `json:"connector_count"`
	LastError      *string    `json:"last_error"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (r *jsonRepository) Scan(value any) error {
	return json.Unmarshal(value.([]byte), r)
}

func (r jsonRepository) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func convertRepositoryToJsonModel(src *pipelineservice.Repository) (*jsonRepository, error) {
	result := &jsonRepository{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.URL = src.URL
	if src.AuthHeader == nil {
		result.AuthHeader = nil
	} else {
		result.AuthHeader = toPtr(fromPtr(src.AuthHeader))
	}
	tmp6, err := convertRepositoryStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp6
	if src.LastSyncedAt == nil {
		result.LastSyncedAt = nil
	} else {
		result.LastSyncedAt = toPtr((fromPtr(src.LastSyncedAt)).UTC())
	}
	result.ConnectorCount = src.ConnectorCount
	if src.LastError == nil {
		result.LastError = nil
	} else {
		result.LastError = toPtr(fromPtr(src.LastError))
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertRepositoryFromJsonModel(src *jsonRepository) (*pipelineservice.Repository, error) {
	result := &pipelineservice.Repository{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.URL = src.URL
	if src.AuthHeader == nil {
		result.AuthHeader = nil
	} else {
		result.AuthHeader = toPtr(fromPtr(src.AuthHeader))
	}
	tmp20, err := convertRepositoryStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp20
	if src.LastSyncedAt == nil {
		result.LastSyncedAt = nil
	} else {
		result.LastSyncedAt = toPtr(fromPtr(src.LastSyncedAt))
	}
	result.ConnectorCount = src.ConnectorCount
	if src.LastError == nil {
		result.LastError = nil
	} else {
		result.LastError = toPtr(fromPtr(src.LastError))
	}
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}

type jsonRepositoryConnector struct {
	ID               uuid.UUID                       `json:"id"`
	RepositoryID     uuid.UUID                       `json:"repository_id"`
	DockerRepository string                          `json:"docker_repository"`
	DockerImageTag   string                          `json:"docker_image_tag"`
	Name             string                          `json:"name"`
	ConnectorType    string                          `json:"connector_type"`
	DocumentationURL string                          `json:"documentation_url"`
	ReleaseStage     string                          `json:"release_stage"`
	IconURL          string                          `json:"icon_url"`
	Spec             jsonb                           `json:"spec"`
	SupportLevel     string                          `json:"support_level"`
	License          string                          `json:"license"`
	SourceType       string                          `json:"source_type"`
	Metadata         jsonRepositoryConnectorMetadata `json:"metadata"`
}

func (r *jsonRepositoryConnector) Scan(value any) error {
	return json.Unmarshal(value.([]byte), r)
}

func (r jsonRepositoryConnector) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func convertRepositoryConnectorToJsonModel(src *pipelineservice.RepositoryConnector) (*jsonRepositoryConnector, error) {
	result := &jsonRepositoryConnector{}
	result.ID = src.ID
	result.RepositoryID = src.RepositoryID
	result.DockerRepository = src.DockerRepository
	result.DockerImageTag = src.DockerImageTag
	result.Name = src.Name
	tmp5, err := convertConnectorTypeToDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp5
	result.DocumentationURL = src.DocumentationURL
	tmp7, err := convertReleaseStageToDB(src.ReleaseStage)
	if err != nil {
		return nil, err
	}
	result.ReleaseStage = tmp7
	result.IconURL = src.IconURL
	result.Spec = src.Spec
	tmp10, err := convertSupportLevelToDB(src.SupportLevel)
	if err != nil {
		return nil, err
	}
	result.SupportLevel = tmp10
	result.License = src.License
	tmp12, err := convertSourceTypeToDB(src.SourceType)
	if err != nil {
		return nil, err
	}
	result.SourceType = tmp12
	tmp13, err := convertRepositoryConnectorMetadataToJsonModel(toPtr(src.Metadata))
	if err != nil {
		return nil, errors.Wrap(err, "convert RepositoryConnectorMetadata to db")
	}
	result.Metadata = *tmp13
	return result, nil
}

func convertRepositoryConnectorFromJsonModel(src *jsonRepositoryConnector) (*pipelineservice.RepositoryConnector, error) {
	result := &pipelineservice.RepositoryConnector{}
	result.ID = src.ID
	result.RepositoryID = src.RepositoryID
	result.DockerRepository = src.DockerRepository
	result.DockerImageTag = src.DockerImageTag
	result.Name = src.Name
	tmp19, err := convertConnectorTypeFromDB(src.ConnectorType)
	if err != nil {
		return nil, err
	}
	result.ConnectorType = tmp19
	result.DocumentationURL = src.DocumentationURL
	tmp21, err := convertReleaseStageFromDB(src.ReleaseStage)
	if err != nil {
		return nil, err
	}
	result.ReleaseStage = tmp21
	result.IconURL = src.IconURL
	result.Spec = src.Spec
	tmp24, err := convertSupportLevelFromDB(src.SupportLevel)
	if err != nil {
		return nil, err
	}
	result.SupportLevel = tmp24
	result.License = src.License
	tmp26, err := convertSourceTypeFromDB(src.SourceType)
	if err != nil {
		return nil, err
	}
	result.SourceType = tmp26
	tmp27, err := convertRepositoryConnectorMetadataFromJsonModel(toPtr(src.Metadata))
	if err != nil {
		return nil, err
	}

	result.Metadata = fromPtr(tmp27)
	return result, nil
}

type jsonRepositoryConnectorMetadata struct {
	BreakingChanges           mapValue[string, jsonBreakingChange]     `json:"breaking_changes"`
	ResourceRequirements      *jsonResourceRequirements                `json:"resource_requirements"`
	MaxSecondsBetweenMessages *int                                     `json:"max_seconds_between_messages"`
	ExternalDocumentationURLs sliceValue[jsonExternalDocumentationURL] `json:"external_documentation_urls"`
	SuggestedStreams          stringSliceValue                         `json:"suggested_streams"`
	AllowedHosts              stringSliceValue                         `json:"allowed_hosts"`
	ErdURL                    string                                   `json:"erd_url"`
	ReleaseDate               string                                   `json:"release_date"`
	Language                  string                                   `json:"language"`
	Tags                      stringSliceValue                         `json:"tags"`
	SupportsRefreshes         bool                                     `json:"supports_refreshes"`
	SupportsFileTransfer      bool                                     `json:"supports_file_transfer"`
	SupportsDataActivation    bool                                     `json:"supports_data_activation"`
	MigrationDocumentationURL string                                   `json:"migration_documentation_url"`
}

func (r *jsonRepositoryConnectorMetadata) Scan(value any) error {
	return json.Unmarshal(value.([]byte), r)
}

func (r jsonRepositoryConnectorMetadata) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func convertRepositoryConnectorMetadataToJsonModel(src *pipelineservice.RepositoryConnectorMetadata) (*jsonRepositoryConnectorMetadata, error) {
	result := &jsonRepositoryConnectorMetadata{}
	tmp := make(mapValue[string, jsonBreakingChange], len(src.BreakingChanges))
	for k, v := range src.BreakingChanges {
		tmp1, err := convertBreakingChangeToJsonModel(toPtr(v))
		if err != nil {
			return nil, errors.Wrap(err, "convert BreakingChange to db")
		}
		tmp[k] = *tmp1
	}
	result.BreakingChanges = tmp
	if src.ResourceRequirements != nil {
		tmp2, err := convertResourceRequirementsToJsonModel(src.ResourceRequirements)
		if err != nil {
			return nil, errors.Wrap(err, "convert ResourceRequirements to db")
		}
		result.ResourceRequirements = tmp2
	} else {
		result.ResourceRequirements = nil
	}
	if src.MaxSecondsBetweenMessages == nil {
		result.MaxSecondsBetweenMessages = nil
	} else {
		result.MaxSecondsBetweenMessages = toPtr(fromPtr(src.MaxSecondsBetweenMessages))
	}
	tmp5 := make(sliceValue[jsonExternalDocumentationURL], 0, len(src.ExternalDocumentationURLs))
	for _, el := range src.ExternalDocumentationURLs {
		tmp6, err := convertExternalDocumentationURLToJsonModel(toPtr(el))
		if err != nil {
			return nil, errors.Wrap(err, "convert ExternalDocumentationURL to db")
		}
		tmp5 = append(tmp5, *tmp6)
	}
	result.ExternalDocumentationURLs = tmp5
	tmp7 := make(stringSliceValue, 0, len(src.SuggestedStreams))
	for _, el := range src.SuggestedStreams {
		tmp7 = append(tmp7, el)
	}
	result.SuggestedStreams = tmp7
	tmp9 := make(stringSliceValue, 0, len(src.AllowedHosts))
	for _, el := range src.AllowedHosts {
		tmp9 = append(tmp9, el)
	}
	result.AllowedHosts = tmp9
	result.ErdURL = src.ErdURL
	result.ReleaseDate = src.ReleaseDate
	result.Language = src.Language
	tmp14 := make(stringSliceValue, 0, len(src.Tags))
	for _, el := range src.Tags {
		tmp14 = append(tmp14, el)
	}
	result.Tags = tmp14
	result.SupportsRefreshes = src.SupportsRefreshes
	result.SupportsFileTransfer = src.SupportsFileTransfer
	result.SupportsDataActivation = src.SupportsDataActivation
	result.MigrationDocumentationURL = src.MigrationDocumentationURL
	return result, nil
}

func convertRepositoryConnectorMetadataFromJsonModel(src *jsonRepositoryConnectorMetadata) (*pipelineservice.RepositoryConnectorMetadata, error) {
	result := &pipelineservice.RepositoryConnectorMetadata{}
	tmp20 := make(map[string]pipelineservice.BreakingChange, len(src.BreakingChanges))
	for k1, v1 := range src.BreakingChanges {
		tmp21, err := convertBreakingChangeFromJsonModel(toPtr(v1))
		if err != nil {
			return nil, err
		}

		tmp20[k1] = fromPtr(tmp21)
	}
	result.BreakingChanges = tmp20
	if src.ResourceRequirements != nil {
		tmp22, err := convertResourceRequirementsFromJsonModel(src.ResourceRequirements)
		if err != nil {
			return nil, err
		}
		result.ResourceRequirements = tmp22
	} else {
		result.ResourceRequirements = nil
	}
	if src.MaxSecondsBetweenMessages == nil {
		result.MaxSecondsBetweenMessages = nil
	} else {
		result.MaxSecondsBetweenMessages = toPtr(fromPtr(src.MaxSecondsBetweenMessages))
	}
	tmp25 := make([]pipelineservice.ExternalDocumentationURL, 0, len(src.ExternalDocumentationURLs))
	for _, el := range src.ExternalDocumentationURLs {

		tmp26, err := convertExternalDocumentationURLFromJsonModel(toPtr(el))
		if err != nil {
			return nil, err
		}

		tmp25 = append(tmp25, fromPtr(tmp26))
	}
	result.ExternalDocumentationURLs = tmp25
	tmp27 := make([]string, 0, len(src.SuggestedStreams))
	for _, el := range src.SuggestedStreams {

		tmp27 = append(tmp27, el)
	}
	result.SuggestedStreams = tmp27
	tmp29 := make([]string, 0, len(src.AllowedHosts))
	for _, el := range src.AllowedHosts {

		tmp29 = append(tmp29, el)
	}
	result.AllowedHosts = tmp29
	result.ErdURL = src.ErdURL
	result.ReleaseDate = src.ReleaseDate
	result.Language = src.Language
	tmp34 := make([]string, 0, len(src.Tags))
	for _, el := range src.Tags {

		tmp34 = append(tmp34, el)
	}
	result.Tags = tmp34
	result.SupportsRefreshes = src.SupportsRefreshes
	result.SupportsFileTransfer = src.SupportsFileTransfer
	result.SupportsDataActivation = src.SupportsDataActivation
	result.MigrationDocumentationURL = src.MigrationDocumentationURL
	return result, nil
}

type jsonBreakingChange struct {
	Message                   string `json:"message"`
	MigrationDocumentationURL string `json:"migration_documentation_url"`
	UpgradeDeadline           string `json:"upgrade_deadline"`
}

func (b *jsonBreakingChange) Scan(value any) error {
	return json.Unmarshal(value.([]byte), b)
}

func (b jsonBreakingChange) Value() (driver.Value, error) {
	return json.Marshal(b)
}

func convertBreakingChangeToJsonModel(src *pipelineservice.BreakingChange) (*jsonBreakingChange, error) {
	result := &jsonBreakingChange{}
	result.Message = src.Message
	result.MigrationDocumentationURL = src.MigrationDocumentationURL
	result.UpgradeDeadline = src.UpgradeDeadline
	return result, nil
}

func convertBreakingChangeFromJsonModel(src *jsonBreakingChange) (*pipelineservice.BreakingChange, error) {
	result := &pipelineservice.BreakingChange{}
	result.Message = src.Message
	result.MigrationDocumentationURL = src.MigrationDocumentationURL
	result.UpgradeDeadline = src.UpgradeDeadline
	return result, nil
}

type jsonResourceRequirements struct {
	JobSpecific sliceValue[jsonJobSpecificResourceRequirement] `json:"job_specific"`
}

func (r *jsonResourceRequirements) Scan(value any) error {
	return json.Unmarshal(value.([]byte), r)
}

func (r jsonResourceRequirements) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func convertResourceRequirementsToJsonModel(src *pipelineservice.ResourceRequirements) (*jsonResourceRequirements, error) {
	result := &jsonResourceRequirements{}
	tmp := make(sliceValue[jsonJobSpecificResourceRequirement], 0, len(src.JobSpecific))
	for _, el := range src.JobSpecific {
		tmp1, err := convertJobSpecificResourceRequirementToJsonModel(toPtr(el))
		if err != nil {
			return nil, errors.Wrap(err, "convert JobSpecificResourceRequirement to db")
		}
		tmp = append(tmp, *tmp1)
	}
	result.JobSpecific = tmp
	return result, nil
}

func convertResourceRequirementsFromJsonModel(src *jsonResourceRequirements) (*pipelineservice.ResourceRequirements, error) {
	result := &pipelineservice.ResourceRequirements{}
	tmp2 := make([]pipelineservice.JobSpecificResourceRequirement, 0, len(src.JobSpecific))
	for _, el := range src.JobSpecific {

		tmp3, err := convertJobSpecificResourceRequirementFromJsonModel(toPtr(el))
		if err != nil {
			return nil, err
		}

		tmp2 = append(tmp2, fromPtr(tmp3))
	}
	result.JobSpecific = tmp2
	return result, nil
}

type jsonJobSpecificResourceRequirement struct {
	JobType              string                        `json:"job_type"`
	ResourceRequirements jsonResourceRequirementValues `json:"resource_requirements"`
}

func (j *jsonJobSpecificResourceRequirement) Scan(value any) error {
	return json.Unmarshal(value.([]byte), j)
}

func (j jsonJobSpecificResourceRequirement) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func convertJobSpecificResourceRequirementToJsonModel(src *pipelineservice.JobSpecificResourceRequirement) (*jsonJobSpecificResourceRequirement, error) {
	result := &jsonJobSpecificResourceRequirement{}
	result.JobType = src.JobType
	tmp1, err := convertResourceRequirementValuesToJsonModel(toPtr(src.ResourceRequirements))
	if err != nil {
		return nil, errors.Wrap(err, "convert ResourceRequirementValues to db")
	}
	result.ResourceRequirements = *tmp1
	return result, nil
}

func convertJobSpecificResourceRequirementFromJsonModel(src *jsonJobSpecificResourceRequirement) (*pipelineservice.JobSpecificResourceRequirement, error) {
	result := &pipelineservice.JobSpecificResourceRequirement{}
	result.JobType = src.JobType
	tmp3, err := convertResourceRequirementValuesFromJsonModel(toPtr(src.ResourceRequirements))
	if err != nil {
		return nil, err
	}

	result.ResourceRequirements = fromPtr(tmp3)
	return result, nil
}

type jsonResourceRequirementValues struct {
	MemoryLimit   string `json:"memory_limit"`
	MemoryRequest string `json:"memory_request"`
	CPULimit      string `json:"cpu_limit"`
	CPURequest    string `json:"cpu_request"`
}

func (r *jsonResourceRequirementValues) Scan(value any) error {
	return json.Unmarshal(value.([]byte), r)
}

func (r jsonResourceRequirementValues) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func convertResourceRequirementValuesToJsonModel(src *pipelineservice.ResourceRequirementValues) (*jsonResourceRequirementValues, error) {
	result := &jsonResourceRequirementValues{}
	result.MemoryLimit = src.MemoryLimit
	result.MemoryRequest = src.MemoryRequest
	result.CPULimit = src.CPULimit
	result.CPURequest = src.CPURequest
	return result, nil
}

func convertResourceRequirementValuesFromJsonModel(src *jsonResourceRequirementValues) (*pipelineservice.ResourceRequirementValues, error) {
	result := &pipelineservice.ResourceRequirementValues{}
	result.MemoryLimit = src.MemoryLimit
	result.MemoryRequest = src.MemoryRequest
	result.CPULimit = src.CPULimit
	result.CPURequest = src.CPURequest
	return result, nil
}

type jsonExternalDocumentationURL struct {
	Title string `json:"title"`
	Type  string `json:"type"`
	URL   string `json:"url"`
}

func (e *jsonExternalDocumentationURL) Scan(value any) error {
	return json.Unmarshal(value.([]byte), e)
}

func (e jsonExternalDocumentationURL) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func convertExternalDocumentationURLToJsonModel(src *pipelineservice.ExternalDocumentationURL) (*jsonExternalDocumentationURL, error) {
	result := &jsonExternalDocumentationURL{}
	result.Title = src.Title
	result.Type = src.Type
	result.URL = src.URL
	return result, nil
}

func convertExternalDocumentationURLFromJsonModel(src *jsonExternalDocumentationURL) (*pipelineservice.ExternalDocumentationURL, error) {
	result := &pipelineservice.ExternalDocumentationURL{}
	result.Title = src.Title
	result.Type = src.Type
	result.URL = src.URL
	return result, nil
}

type jsonSource struct {
	ID                 uuid.UUID `json:"id"`
	WorkspaceID        uuid.UUID `json:"workspace_id"`
	Name               string    `json:"name"`
	ManagedConnectorID uuid.UUID `json:"managed_connector_id"`
	Config             jsonb     `json:"config"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	RuntimeConfig      *jsonb    `json:"runtime_config"`
}

func (s *jsonSource) Scan(value any) error {
	return json.Unmarshal(value.([]byte), s)
}

func (s jsonSource) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func convertSourceToJsonModel(src *pipelineservice.Source) (*jsonSource, error) {
	result := &jsonSource{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}

func convertSourceFromJsonModel(src *jsonSource) (*pipelineservice.Source, error) {
	result := &pipelineservice.Source{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}

type jsonDestination struct {
	ID                 uuid.UUID `json:"id"`
	WorkspaceID        uuid.UUID `json:"workspace_id"`
	Name               string    `json:"name"`
	ManagedConnectorID uuid.UUID `json:"managed_connector_id"`
	Config             jsonb     `json:"config"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	RuntimeConfig      *jsonb    `json:"runtime_config"`
}

func (d *jsonDestination) Scan(value any) error {
	return json.Unmarshal(value.([]byte), d)
}

func (d jsonDestination) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func convertDestinationToJsonModel(src *pipelineservice.Destination) (*jsonDestination, error) {
	result := &jsonDestination{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}

func convertDestinationFromJsonModel(src *jsonDestination) (*pipelineservice.Destination, error) {
	result := &pipelineservice.Destination{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	result.ManagedConnectorID = src.ManagedConnectorID
	result.Config = src.Config
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.RuntimeConfig == nil {
		result.RuntimeConfig = nil
	} else {
		result.RuntimeConfig = toPtr(fromPtr(src.RuntimeConfig))
	}
	return result, nil
}

type jsonConnection struct {
	ID                    uuid.UUID  `json:"id"`
	WorkspaceID           uuid.UUID  `json:"workspace_id"`
	Name                  string     `json:"name"`
	Status                string     `json:"status"`
	SourceID              uuid.UUID  `json:"source_id"`
	DestinationID         uuid.UUID  `json:"destination_id"`
	Schedule              *string    `json:"schedule"`
	SchemaChangePolicy    string     `json:"schema_change_policy"`
	MaxAttempts           int        `json:"max_attempts"`
	NamespaceDefinition   string     `json:"namespace_definition"`
	CustomNamespaceFormat *string    `json:"custom_namespace_format"`
	StreamPrefix          *string    `json:"stream_prefix"`
	NextScheduledAt       *time.Time `json:"next_scheduled_at"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (c *jsonConnection) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonConnection) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertConnectionToJsonModel(src *pipelineservice.Connection) (*jsonConnection, error) {
	result := &jsonConnection{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	tmp3, err := convertConnectionStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp3
	result.SourceID = src.SourceID
	result.DestinationID = src.DestinationID
	if src.Schedule == nil {
		result.Schedule = nil
	} else {
		result.Schedule = toPtr(fromPtr(src.Schedule))
	}
	tmp8, err := convertSchemaChangePolicyToDB(src.SchemaChangePolicy)
	if err != nil {
		return nil, err
	}
	result.SchemaChangePolicy = tmp8
	result.MaxAttempts = src.MaxAttempts
	tmp10, err := convertNamespaceDefinitionToDB(src.NamespaceDefinition)
	if err != nil {
		return nil, err
	}
	result.NamespaceDefinition = tmp10
	if src.CustomNamespaceFormat == nil {
		result.CustomNamespaceFormat = nil
	} else {
		result.CustomNamespaceFormat = toPtr(fromPtr(src.CustomNamespaceFormat))
	}
	if src.StreamPrefix == nil {
		result.StreamPrefix = nil
	} else {
		result.StreamPrefix = toPtr(fromPtr(src.StreamPrefix))
	}
	if src.NextScheduledAt == nil {
		result.NextScheduledAt = nil
	} else {
		result.NextScheduledAt = toPtr((fromPtr(src.NextScheduledAt)).UTC())
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertConnectionFromJsonModel(src *jsonConnection) (*pipelineservice.Connection, error) {
	result := &pipelineservice.Connection{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	tmp22, err := convertConnectionStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp22
	result.SourceID = src.SourceID
	result.DestinationID = src.DestinationID
	if src.Schedule == nil {
		result.Schedule = nil
	} else {
		result.Schedule = toPtr(fromPtr(src.Schedule))
	}
	tmp27, err := convertSchemaChangePolicyFromDB(src.SchemaChangePolicy)
	if err != nil {
		return nil, err
	}
	result.SchemaChangePolicy = tmp27
	result.MaxAttempts = src.MaxAttempts
	tmp29, err := convertNamespaceDefinitionFromDB(src.NamespaceDefinition)
	if err != nil {
		return nil, err
	}
	result.NamespaceDefinition = tmp29
	if src.CustomNamespaceFormat == nil {
		result.CustomNamespaceFormat = nil
	} else {
		result.CustomNamespaceFormat = toPtr(fromPtr(src.CustomNamespaceFormat))
	}
	if src.StreamPrefix == nil {
		result.StreamPrefix = nil
	} else {
		result.StreamPrefix = toPtr(fromPtr(src.StreamPrefix))
	}
	if src.NextScheduledAt == nil {
		result.NextScheduledAt = nil
	} else {
		result.NextScheduledAt = toPtr(fromPtr(src.NextScheduledAt))
	}
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}

type jsonJob struct {
	ID            uuid.UUID  `json:"id"`
	ConnectionID  uuid.UUID  `json:"connection_id"`
	Status        string     `json:"status"`
	JobType       string     `json:"job_type"`
	ScheduledAt   time.Time  `json:"scheduled_at"`
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	Error         *string    `json:"error"`
	Attempt       int        `json:"attempt"`
	MaxAttempts   int        `json:"max_attempts"`
	WorkerID      *string    `json:"worker_id"`
	HeartbeatAt   *time.Time `json:"heartbeat_at"`
	K8sJobName    *string    `json:"k8s_job_name"`
	FailureReason *string    `json:"failure_reason"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (j *jsonJob) Scan(value any) error {
	return json.Unmarshal(value.([]byte), j)
}

func (j jsonJob) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func convertJobToJsonModel(src *pipelineservice.Job) (*jsonJob, error) {
	result := &jsonJob{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	tmp2, err := convertJobStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp2
	tmp3, err := convertJobTypeToDB(src.JobType)
	if err != nil {
		return nil, err
	}
	result.JobType = tmp3
	result.ScheduledAt = (src.ScheduledAt).UTC()
	if src.StartedAt == nil {
		result.StartedAt = nil
	} else {
		result.StartedAt = toPtr((fromPtr(src.StartedAt)).UTC())
	}
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr((fromPtr(src.CompletedAt)).UTC())
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.Attempt = src.Attempt
	result.MaxAttempts = src.MaxAttempts
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	if src.HeartbeatAt == nil {
		result.HeartbeatAt = nil
	} else {
		result.HeartbeatAt = toPtr((fromPtr(src.HeartbeatAt)).UTC())
	}
	if src.K8sJobName == nil {
		result.K8sJobName = nil
	} else {
		result.K8sJobName = toPtr(fromPtr(src.K8sJobName))
	}
	if src.FailureReason == nil {
		result.FailureReason = nil
	} else {
		result.FailureReason = toPtr(fromPtr(src.FailureReason))
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertJobFromJsonModel(src *jsonJob) (*pipelineservice.Job, error) {
	result := &pipelineservice.Job{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	tmp24, err := convertJobStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp24
	tmp25, err := convertJobTypeFromDB(src.JobType)
	if err != nil {
		return nil, err
	}
	result.JobType = tmp25
	result.ScheduledAt = src.ScheduledAt
	if src.StartedAt == nil {
		result.StartedAt = nil
	} else {
		result.StartedAt = toPtr(fromPtr(src.StartedAt))
	}
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr(fromPtr(src.CompletedAt))
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.Attempt = src.Attempt
	result.MaxAttempts = src.MaxAttempts
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	if src.HeartbeatAt == nil {
		result.HeartbeatAt = nil
	} else {
		result.HeartbeatAt = toPtr(fromPtr(src.HeartbeatAt))
	}
	if src.K8sJobName == nil {
		result.K8sJobName = nil
	} else {
		result.K8sJobName = toPtr(fromPtr(src.K8sJobName))
	}
	if src.FailureReason == nil {
		result.FailureReason = nil
	} else {
		result.FailureReason = toPtr(fromPtr(src.FailureReason))
	}
	result.CreatedAt = src.CreatedAt
	return result, nil
}

type jsonJobAttempt struct {
	ID            uuid.UUID  `json:"id"`
	JobID         uuid.UUID  `json:"job_id"`
	AttemptNumber int        `json:"attempt_number"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	Error         *string    `json:"error"`
	SyncStatsJSON jsonb      `json:"sync_stats_json"`
}

func (j *jsonJobAttempt) Scan(value any) error {
	return json.Unmarshal(value.([]byte), j)
}

func (j jsonJobAttempt) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func convertJobAttemptToJsonModel(src *pipelineservice.JobAttempt) (*jsonJobAttempt, error) {
	result := &jsonJobAttempt{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.AttemptNumber = src.AttemptNumber
	result.StartedAt = (src.StartedAt).UTC()
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr((fromPtr(src.CompletedAt)).UTC())
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.SyncStatsJSON = src.SyncStatsJSON
	return result, nil
}

func convertJobAttemptFromJsonModel(src *jsonJobAttempt) (*pipelineservice.JobAttempt, error) {
	result := &pipelineservice.JobAttempt{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.AttemptNumber = src.AttemptNumber
	result.StartedAt = src.StartedAt
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr(fromPtr(src.CompletedAt))
	}
	if src.Error == nil {
		result.Error = nil
	} else {
		result.Error = toPtr(fromPtr(src.Error))
	}
	result.SyncStatsJSON = src.SyncStatsJSON
	return result, nil
}

type jsonCatalogDiscovery struct {
	ID           uuid.UUID `json:"id"`
	SourceID     uuid.UUID `json:"source_id"`
	Version      int       `json:"version"`
	CatalogJSON  jsonb     `json:"catalog_json"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

func (c *jsonCatalogDiscovery) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonCatalogDiscovery) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertCatalogDiscoveryToJsonModel(src *pipelineservice.CatalogDiscovery) (*jsonCatalogDiscovery, error) {
	result := &jsonCatalogDiscovery{}
	result.ID = src.ID
	result.SourceID = src.SourceID
	result.Version = src.Version
	result.CatalogJSON = src.CatalogJSON
	result.DiscoveredAt = (src.DiscoveredAt).UTC()
	return result, nil
}

func convertCatalogDiscoveryFromJsonModel(src *jsonCatalogDiscovery) (*pipelineservice.CatalogDiscovery, error) {
	result := &pipelineservice.CatalogDiscovery{}
	result.ID = src.ID
	result.SourceID = src.SourceID
	result.Version = src.Version
	result.CatalogJSON = src.CatalogJSON
	result.DiscoveredAt = src.DiscoveredAt
	return result, nil
}

type jsonConfiguredCatalog struct {
	ID           uuid.UUID `json:"id"`
	ConnectionID uuid.UUID `json:"connection_id"`
	StreamsJSON  jsonb     `json:"streams_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (c *jsonConfiguredCatalog) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonConfiguredCatalog) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertConfiguredCatalogToJsonModel(src *pipelineservice.ConfiguredCatalog) (*jsonConfiguredCatalog, error) {
	result := &jsonConfiguredCatalog{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	result.StreamsJSON = src.StreamsJSON
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertConfiguredCatalogFromJsonModel(src *jsonConfiguredCatalog) (*pipelineservice.ConfiguredCatalog, error) {
	result := &pipelineservice.ConfiguredCatalog{}
	result.ID = src.ID
	result.ConnectionID = src.ConnectionID
	result.StreamsJSON = src.StreamsJSON
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}

type jsonJobLog struct {
	ID        int64     `json:"id"`
	JobID     uuid.UUID `json:"job_id"`
	LogLine   string    `json:"log_line"`
	CreatedAt time.Time `json:"created_at"`
}

func (j *jsonJobLog) Scan(value any) error {
	return json.Unmarshal(value.([]byte), j)
}

func (j jsonJobLog) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func convertJobLogToJsonModel(src *pipelineservice.JobLog) (*jsonJobLog, error) {
	result := &jsonJobLog{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.LogLine = src.LogLine
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertJobLogFromJsonModel(src *jsonJobLog) (*pipelineservice.JobLog, error) {
	result := &pipelineservice.JobLog{}
	result.ID = src.ID
	result.JobID = src.JobID
	result.LogLine = src.LogLine
	result.CreatedAt = src.CreatedAt
	return result, nil
}

type jsonConnectionState struct {
	ConnectionID uuid.UUID `json:"connection_id"`
	StateType    string    `json:"state_type"`
	StateBlob    jsonb     `json:"state_blob"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (c *jsonConnectionState) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonConnectionState) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertConnectionStateToJsonModel(src *pipelineservice.ConnectionState) (*jsonConnectionState, error) {
	result := &jsonConnectionState{}
	result.ConnectionID = src.ConnectionID
	result.StateType = src.StateType
	result.StateBlob = src.StateBlob
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertConnectionStateFromJsonModel(src *jsonConnectionState) (*pipelineservice.ConnectionState, error) {
	result := &pipelineservice.ConnectionState{}
	result.ConnectionID = src.ConnectionID
	result.StateType = src.StateType
	result.StateBlob = src.StateBlob
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}

type jsonCheckPayload struct {
	SourceID           *uuid.UUID `json:"source_id"`
	DestinationID      *uuid.UUID `json:"destination_id"`
	ManagedConnectorID uuid.UUID  `json:"managed_connector_id"`
	Config             *string    `json:"config"`
}

func (c *jsonCheckPayload) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonCheckPayload) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertCheckPayloadToJsonModel(src *pipelineservice.CheckPayload) (*jsonCheckPayload, error) {
	result := &jsonCheckPayload{}
	if src.SourceID == nil {
		result.SourceID = nil
	} else {
		result.SourceID = toPtr(fromPtr(src.SourceID))
	}
	if src.DestinationID == nil {
		result.DestinationID = nil
	} else {
		result.DestinationID = toPtr(fromPtr(src.DestinationID))
	}
	result.ManagedConnectorID = src.ManagedConnectorID
	if src.Config == nil {
		result.Config = nil
	} else {
		result.Config = toPtr(fromPtr(src.Config))
	}
	return result, nil
}

func convertCheckPayloadFromJsonModel(src *jsonCheckPayload) (*pipelineservice.CheckPayload, error) {
	result := &pipelineservice.CheckPayload{}
	if src.SourceID == nil {
		result.SourceID = nil
	} else {
		result.SourceID = toPtr(fromPtr(src.SourceID))
	}
	if src.DestinationID == nil {
		result.DestinationID = nil
	} else {
		result.DestinationID = toPtr(fromPtr(src.DestinationID))
	}
	result.ManagedConnectorID = src.ManagedConnectorID
	if src.Config == nil {
		result.Config = nil
	} else {
		result.Config = toPtr(fromPtr(src.Config))
	}
	return result, nil
}

type jsonSpecPayload struct {
	ManagedConnectorID uuid.UUID `json:"managed_connector_id"`
}

func (s *jsonSpecPayload) Scan(value any) error {
	return json.Unmarshal(value.([]byte), s)
}

func (s jsonSpecPayload) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func convertSpecPayloadToJsonModel(src *pipelineservice.SpecPayload) (*jsonSpecPayload, error) {
	result := &jsonSpecPayload{}
	result.ManagedConnectorID = src.ManagedConnectorID
	return result, nil
}

func convertSpecPayloadFromJsonModel(src *jsonSpecPayload) (*pipelineservice.SpecPayload, error) {
	result := &pipelineservice.SpecPayload{}
	result.ManagedConnectorID = src.ManagedConnectorID
	return result, nil
}

type jsonDiscoverPayload struct {
	SourceID           uuid.UUID `json:"source_id"`
	ManagedConnectorID uuid.UUID `json:"managed_connector_id"`
}

func (d *jsonDiscoverPayload) Scan(value any) error {
	return json.Unmarshal(value.([]byte), d)
}

func (d jsonDiscoverPayload) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func convertDiscoverPayloadToJsonModel(src *pipelineservice.DiscoverPayload) (*jsonDiscoverPayload, error) {
	result := &jsonDiscoverPayload{}
	result.SourceID = src.SourceID
	result.ManagedConnectorID = src.ManagedConnectorID
	return result, nil
}

func convertDiscoverPayloadFromJsonModel(src *jsonDiscoverPayload) (*pipelineservice.DiscoverPayload, error) {
	result := &pipelineservice.DiscoverPayload{}
	result.SourceID = src.SourceID
	result.ManagedConnectorID = src.ManagedConnectorID
	return result, nil
}

type jsonCheckResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (c *jsonCheckResult) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonCheckResult) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertCheckResultToJsonModel(src *pipelineservice.CheckResult) (*jsonCheckResult, error) {
	result := &jsonCheckResult{}
	result.Success = src.Success
	result.Message = src.Message
	return result, nil
}

func convertCheckResultFromJsonModel(src *jsonCheckResult) (*pipelineservice.CheckResult, error) {
	result := &pipelineservice.CheckResult{}
	result.Success = src.Success
	result.Message = src.Message
	return result, nil
}

type jsonSpecResult struct {
	DocumentationURL              string `json:"documentation_url"`
	ChangelogURL                  string `json:"changelog_url"`
	ConnectionSpecification       jsonb  `json:"connection_specification"`
	SupportsIncremental           bool   `json:"supports_incremental"`
	SupportsNormalization         bool   `json:"supports_normalization"`
	SupportsDBT                   bool   `json:"supports_dbt"`
	SupportedDestinationSyncModes jsonb  `json:"supported_destination_sync_modes"`
	AdvancedAuth                  jsonb  `json:"advanced_auth"`
	ProtocolVersion               string `json:"protocol_version"`
}

func (s *jsonSpecResult) Scan(value any) error {
	return json.Unmarshal(value.([]byte), s)
}

func (s jsonSpecResult) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func convertSpecResultToJsonModel(src *pipelineservice.SpecResult) (*jsonSpecResult, error) {
	result := &jsonSpecResult{}
	result.DocumentationURL = src.DocumentationURL
	result.ChangelogURL = src.ChangelogURL
	result.ConnectionSpecification = src.ConnectionSpecification
	result.SupportsIncremental = src.SupportsIncremental
	result.SupportsNormalization = src.SupportsNormalization
	result.SupportsDBT = src.SupportsDBT
	result.SupportedDestinationSyncModes = src.SupportedDestinationSyncModes
	result.AdvancedAuth = src.AdvancedAuth
	result.ProtocolVersion = src.ProtocolVersion
	return result, nil
}

func convertSpecResultFromJsonModel(src *jsonSpecResult) (*pipelineservice.SpecResult, error) {
	result := &pipelineservice.SpecResult{}
	result.DocumentationURL = src.DocumentationURL
	result.ChangelogURL = src.ChangelogURL
	result.ConnectionSpecification = src.ConnectionSpecification
	result.SupportsIncremental = src.SupportsIncremental
	result.SupportsNormalization = src.SupportsNormalization
	result.SupportsDBT = src.SupportsDBT
	result.SupportedDestinationSyncModes = src.SupportedDestinationSyncModes
	result.AdvancedAuth = src.AdvancedAuth
	result.ProtocolVersion = src.ProtocolVersion
	return result, nil
}

type jsonDiscoverResult struct {
	Catalog string `json:"catalog"`
}

func (d *jsonDiscoverResult) Scan(value any) error {
	return json.Unmarshal(value.([]byte), d)
}

func (d jsonDiscoverResult) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func convertDiscoverResultToJsonModel(src *pipelineservice.DiscoverResult) (*jsonDiscoverResult, error) {
	result := &jsonDiscoverResult{}
	result.Catalog = src.Catalog
	return result, nil
}

func convertDiscoverResultFromJsonModel(src *jsonDiscoverResult) (*pipelineservice.DiscoverResult, error) {
	result := &pipelineservice.DiscoverResult{}
	result.Catalog = src.Catalog
	return result, nil
}

type jsonWorkspaceSettings struct {
	WorkspaceID         uuid.UUID `json:"workspace_id"`
	MaxJobsPerWorkspace int       `json:"max_jobs_per_workspace"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (w *jsonWorkspaceSettings) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceSettings) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceSettingsToJsonModel(src *pipelineservice.WorkspaceSettings) (*jsonWorkspaceSettings, error) {
	result := &jsonWorkspaceSettings{}
	result.WorkspaceID = src.WorkspaceID
	result.MaxJobsPerWorkspace = src.MaxJobsPerWorkspace
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertWorkspaceSettingsFromJsonModel(src *jsonWorkspaceSettings) (*pipelineservice.WorkspaceSettings, error) {
	result := &pipelineservice.WorkspaceSettings{}
	result.WorkspaceID = src.WorkspaceID
	result.MaxJobsPerWorkspace = src.MaxJobsPerWorkspace
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}

type jsonConnectorTask struct {
	ID           uuid.UUID                 `json:"id"`
	WorkspaceID  uuid.UUID                 `json:"workspace_id"`
	TaskType     string                    `json:"task_type"`
	Status       string                    `json:"status"`
	Payload      *jsonConnectorTaskPayload `json:"payload"`
	Result       *jsonConnectorTaskResult  `json:"result"`
	ErrorMessage *string                   `json:"error_message"`
	WorkerID     *string                   `json:"worker_id"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	CompletedAt  *time.Time                `json:"completed_at"`
}

func (c *jsonConnectorTask) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonConnectorTask) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertConnectorTaskToJsonModel(src *pipelineservice.ConnectorTask) (*jsonConnectorTask, error) {
	result := &jsonConnectorTask{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	tmp2, err := convertConnectorTaskTypeToDB(src.TaskType)
	if err != nil {
		return nil, err
	}
	result.TaskType = tmp2
	tmp3, err := convertConnectorTaskStatusToDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp3
	tmp4, err := convertConnectorTaskPayloadToDB(src.Payload)
	if err != nil {
		return nil, err
	}
	result.Payload = tmp4
	if src.Result != nil {
		tmp5, err := convertConnectorTaskResultToDB(*src.Result)
		if err != nil {
			return nil, err
		}
		result.Result = tmp5
	} else {
		result.Result = nil
	}
	if src.ErrorMessage == nil {
		result.ErrorMessage = nil
	} else {
		result.ErrorMessage = toPtr(fromPtr(src.ErrorMessage))
	}
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr((fromPtr(src.CompletedAt)).UTC())
	}
	return result, nil
}

func convertConnectorTaskFromJsonModel(src *jsonConnectorTask) (*pipelineservice.ConnectorTask, error) {
	result := &pipelineservice.ConnectorTask{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	tmp16, err := convertConnectorTaskTypeFromDB(src.TaskType)
	if err != nil {
		return nil, err
	}
	result.TaskType = tmp16
	tmp17, err := convertConnectorTaskStatusFromDB(src.Status)
	if err != nil {
		return nil, err
	}
	result.Status = tmp17
	tmp18, err := convertConnectorTaskPayloadFromDB(src.Payload)
	if err != nil {
		return nil, fmt.Errorf("convert ConnectorTaskPayload to service type: %w", err)
	}
	result.Payload = tmp18
	if src.Result != nil {
		tmp19, err := convertConnectorTaskResultFromDB(src.Result)
		if err != nil {
			return nil, fmt.Errorf("convert ConnectorTaskResult to service type: %w", err)
		}
		result.Result = toPtr(tmp19)
	} else {
		result.Result = nil
	}
	if src.ErrorMessage == nil {
		result.ErrorMessage = nil
	} else {
		result.ErrorMessage = toPtr(fromPtr(src.ErrorMessage))
	}
	if src.WorkerID == nil {
		result.WorkerID = nil
	} else {
		result.WorkerID = toPtr(fromPtr(src.WorkerID))
	}
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt
	if src.CompletedAt == nil {
		result.CompletedAt = nil
	} else {
		result.CompletedAt = toPtr(fromPtr(src.CompletedAt))
	}
	return result, nil
}

type jsonStreamGeneration struct {
	ConnectionID    uuid.UUID `json:"connection_id"`
	StreamNamespace string    `json:"stream_namespace"`
	StreamName      string    `json:"stream_name"`
	GenerationID    int64     `json:"generation_id"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (s *jsonStreamGeneration) Scan(value any) error {
	return json.Unmarshal(value.([]byte), s)
}

func (s jsonStreamGeneration) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func convertStreamGenerationToJsonModel(src *pipelineservice.StreamGeneration) (*jsonStreamGeneration, error) {
	result := &jsonStreamGeneration{}
	result.ConnectionID = src.ConnectionID
	result.StreamNamespace = src.StreamNamespace
	result.StreamName = src.StreamName
	result.GenerationID = src.GenerationID
	result.UpdatedAt = (src.UpdatedAt).UTC()
	return result, nil
}

func convertStreamGenerationFromJsonModel(src *jsonStreamGeneration) (*pipelineservice.StreamGeneration, error) {
	result := &pipelineservice.StreamGeneration{}
	result.ConnectionID = src.ConnectionID
	result.StreamNamespace = src.StreamNamespace
	result.StreamName = src.StreamName
	result.GenerationID = src.GenerationID
	result.UpdatedAt = src.UpdatedAt
	return result, nil
}
