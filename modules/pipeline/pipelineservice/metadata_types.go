package pipelineservice

import "encoding/json"

// RepositoryConnectorMetadata holds all registry metadata fields stored as JSONB.
// These fields are read on demand (e.g., during connector updates, setup drawer)
// rather than used for listing/filtering queries.
type RepositoryConnectorMetadata struct {
	BreakingChanges           map[string]BreakingChange  `json:"breakingChanges,omitempty"`
	ResourceRequirements      *ResourceRequirements      `json:"resourceRequirements,omitempty"`
	MaxSecondsBetweenMessages *int                       `json:"maxSecondsBetweenMessages,omitempty"`
	ExternalDocumentationURLs []ExternalDocumentationURL `json:"externalDocumentationUrls,omitempty"`
	SuggestedStreams          []string                   `json:"suggestedStreams,omitempty"`
	AllowedHosts              []string                   `json:"allowedHosts,omitempty"`
	ErdURL                    string                     `json:"erdUrl,omitempty"`
	ReleaseDate               string                     `json:"releaseDate,omitempty"`
	Language                  string                     `json:"language,omitempty"`
	Tags                      []string                   `json:"tags,omitempty"`
	SupportsRefreshes         bool                       `json:"supportsRefreshes,omitempty"`
	SupportsFileTransfer      bool                       `json:"supportsFileTransfer,omitempty"`
	SupportsDataActivation    bool                       `json:"supportsDataActivation,omitempty"`
	MigrationDocumentationURL string                     `json:"migrationDocumentationUrl,omitempty"`
}

// BreakingChange represents a single version's breaking change entry.
type BreakingChange struct {
	Message                   string `json:"message"`
	MigrationDocumentationURL string `json:"migrationDocumentationUrl,omitempty"`
	UpgradeDeadline           string `json:"upgradeDeadline,omitempty"`
}

// ResourceRequirements represents connector resource requirements.
type ResourceRequirements struct {
	JobSpecific []JobSpecificResourceRequirement `json:"jobSpecific,omitempty"`
}

// JobSpecificResourceRequirement represents resource requirements for a specific job type.
type JobSpecificResourceRequirement struct {
	JobType              string                    `json:"jobType"`
	ResourceRequirements ResourceRequirementValues `json:"resourceRequirements"`
}

// ResourceRequirementValues holds memory/CPU limit and request values.
type ResourceRequirementValues struct {
	MemoryLimit   string `json:"memory_limit,omitempty"`
	MemoryRequest string `json:"memory_request,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`
	CPURequest    string `json:"cpu_request,omitempty"`
}

// ExternalDocumentationURL represents a documentation link with type and title.
type ExternalDocumentationURL struct {
	Title string `json:"title"`
	Type  string `json:"type"`
	URL   string `json:"url"`
}

// MarshalMetadata serializes RepositoryConnectorMetadata to a JSON string for JSONB storage.
func MarshalMetadata(m *RepositoryConnectorMetadata) string {
	if m == nil {
		return "{}"
	}

	data, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}

	return string(data)
}

// UnmarshalMetadata deserializes a JSON string from JSONB storage into RepositoryConnectorMetadata.
func UnmarshalMetadata(s string) (*RepositoryConnectorMetadata, error) {
	if s == "" || s == "{}" {
		return &RepositoryConnectorMetadata{}, nil
	}

	var m RepositoryConnectorMetadata
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}

	return &m, nil
}
