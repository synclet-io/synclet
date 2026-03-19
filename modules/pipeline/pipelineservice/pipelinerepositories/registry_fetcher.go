package pipelinerepositories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/multierr"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ConnectorData holds parsed connector information from a registry response.
type ConnectorData struct {
	Name             string
	DockerRepository string
	DockerImageTag   string
	DocumentationURL string
	ReleaseStage     string
	IconURL          string
	ConnectorType    string // "source" or "destination"
	Spec             string // JSON string of connectionSpecification, empty if not available
	SupportLevel     string
	License          string
	SourceType       string
	Metadata         string // JSON string of RepositoryConnectorMetadata
}

// RegistryFetcher fetches and parses Airbyte-format connector registry JSON.
type RegistryFetcher struct {
	httpClient *http.Client
}

// NewRegistryFetcher creates a new RegistryFetcher with a default HTTP client.
func NewRegistryFetcher() *RegistryFetcher {
	return &RegistryFetcher{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// airbyteRegistry represents the top-level structure of the Airbyte OSS registry JSON.
type airbyteRegistry struct {
	Sources      []airbyteConnector `json:"sources"`
	Destinations []airbyteConnector `json:"destinations"`
}

// airbyteSpec represents the spec section of a connector entry.
type airbyteSpec struct {
	ConnectionSpecification json.RawMessage `json:"connectionSpecification"`
}

// airbyteReleases represents the releases section of a connector entry.
type airbyteReleases struct {
	BreakingChanges           map[string]airbyteBreakingChange `json:"breakingChanges,omitempty"`
	ReleaseCandidates         json.RawMessage                  `json:"releaseCandidates,omitempty"`
	MigrationDocumentationURL string                           `json:"migrationDocumentationUrl,omitempty"`
}

// airbyteBreakingChange represents a single breaking change entry.
type airbyteBreakingChange struct {
	Message                   string `json:"message"`
	MigrationDocumentationURL string `json:"migrationDocumentationUrl,omitempty"`
	UpgradeDeadline           string `json:"upgradeDeadline,omitempty"`
}

// airbyteResourceRequirements represents the resourceRequirements section.
type airbyteResourceRequirements struct {
	JobSpecific []struct {
		JobType              string `json:"jobType"`
		ResourceRequirements struct {
			MemoryLimit   string `json:"memory_limit,omitempty"`
			MemoryRequest string `json:"memory_request,omitempty"`
			CPULimit      string `json:"cpu_limit,omitempty"`
			CPURequest    string `json:"cpu_request,omitempty"`
		} `json:"resourceRequirements"`
	} `json:"jobSpecific,omitempty"`
}

// airbyteExternalDocURL represents an external documentation URL entry.
type airbyteExternalDocURL struct {
	Title string `json:"title"`
	Type  string `json:"type"`
	URL   string `json:"url"`
}

// airbyteSuggestedStreams represents the suggestedStreams section.
type airbyteSuggestedStreams struct {
	Streams []string `json:"streams"`
}

// airbyteAllowedHosts represents the allowedHosts section.
type airbyteAllowedHosts struct {
	Hosts []string `json:"hosts"`
}

// airbyteConnector represents a single connector entry in the Airbyte registry.
type airbyteConnector struct {
	Name                      string                       `json:"name"`
	DockerRepository          string                       `json:"dockerRepository"`
	DockerImageTag            string                       `json:"dockerImageTag"`
	DocumentationURL          string                       `json:"documentationUrl"`
	ReleaseStage              string                       `json:"releaseStage"`
	Icon                      string                       `json:"icon"`
	IconURL                   string                       `json:"iconUrl"`
	Tombstone                 bool                         `json:"tombstone"`
	Spec                      *airbyteSpec                 `json:"spec"`
	SupportLevel              string                       `json:"supportLevel"`
	License                   string                       `json:"license"`
	SourceType                string                       `json:"sourceType"`
	Releases                  *airbyteReleases             `json:"releases"`
	ResourceRequirements      *airbyteResourceRequirements `json:"resourceRequirements"`
	MaxSecondsBetweenMessages *int                         `json:"maxSecondsBetweenMessages"`
	ExternalDocumentationURLs []airbyteExternalDocURL      `json:"externalDocumentationUrls"`
	SuggestedStreams          *airbyteSuggestedStreams     `json:"suggestedStreams"`
	AllowedHosts              *airbyteAllowedHosts         `json:"allowedHosts"`
	ErdURL                    string                       `json:"erdUrl"`
	ReleaseDate               string                       `json:"releaseDate"`
	Language                  string                       `json:"language"`
	Tags                      []string                     `json:"tags"`
	SupportsRefreshes         bool                         `json:"supportsRefreshes"`
	SupportsFileTransfer      bool                         `json:"supportsFileTransfer"`
	SupportsDataActivation    bool                         `json:"supportsDataActivation"`
}

// Fetch downloads and parses a registry JSON from the given URL,
// filtering out tombstoned and Airbyte Enterprise licensed entries.
func (f *RegistryFetcher) Fetch(ctx context.Context, url string, authHeader *string) (_ []ConnectorData, rerr error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if authHeader != nil && *authHeader != "" {
		req.Header.Set("Authorization", *authHeader)
	}

	resp, err := f.httpClient.Do(req) //nolint:bodyclose // closed via multierr.AppendInvoke below
	if err != nil {
		return nil, fmt.Errorf("fetching registry: %w", err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(resp.Body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	// Limit response size to 50MB to prevent memory exhaustion.
	limitedBody := http.MaxBytesReader(nil, resp.Body, 50*1024*1024)

	var registry airbyteRegistry
	if err := json.NewDecoder(limitedBody).Decode(&registry); err != nil {
		return nil, fmt.Errorf("decoding registry JSON: %w", err)
	}

	var connectors []ConnectorData
	for _, s := range registry.Sources {
		if s.Tombstone || isEnterpriseLicensed(s) {
			continue
		}
		connectors = append(connectors, toConnectorData(s, "source"))
	}
	for _, d := range registry.Destinations {
		if d.Tombstone || isEnterpriseLicensed(d) {
			continue
		}
		connectors = append(connectors, toConnectorData(d, "destination"))
	}
	return connectors, nil
}

// isEnterpriseLicensed returns true if the connector requires an Airbyte Enterprise license.
// These connectors do not function in OSS/self-hosted mode, so they are filtered out (D-03).
func isEnterpriseLicensed(c airbyteConnector) bool {
	return c.License == "Airbyte Enterprise"
}

// toConnectorData maps an airbyteConnector to ConnectorData, extracting all metadata fields.
func toConnectorData(c airbyteConnector, connectorType string) ConnectorData {
	metadata := buildMetadata(c)
	return ConnectorData{
		Name:             c.Name,
		DockerRepository: c.DockerRepository,
		DockerImageTag:   c.DockerImageTag,
		DocumentationURL: c.DocumentationURL,
		ReleaseStage:     c.ReleaseStage,
		IconURL:          c.IconURL, // Use iconUrl CDN URL (D-04), not icon filename
		ConnectorType:    connectorType,
		Spec:             marshalConnectorSpec(c.Spec, c.DocumentationURL),
		SupportLevel:     c.SupportLevel,
		License:          c.License,
		SourceType:       c.SourceType,
		Metadata:         pipelineservice.MarshalMetadata(&metadata),
	}
}

// buildMetadata constructs a RepositoryConnectorMetadata from an airbyteConnector.
func buildMetadata(c airbyteConnector) pipelineservice.RepositoryConnectorMetadata {
	m := pipelineservice.RepositoryConnectorMetadata{
		MaxSecondsBetweenMessages: c.MaxSecondsBetweenMessages,
		ErdURL:                    c.ErdURL,
		ReleaseDate:               c.ReleaseDate,
		Language:                  c.Language,
		Tags:                      c.Tags,
		SupportsRefreshes:         c.SupportsRefreshes,
		SupportsFileTransfer:      c.SupportsFileTransfer,
		SupportsDataActivation:    c.SupportsDataActivation,
	}

	// Breaking changes from releases section.
	if c.Releases != nil {
		if len(c.Releases.BreakingChanges) > 0 {
			m.BreakingChanges = make(map[string]pipelineservice.BreakingChange, len(c.Releases.BreakingChanges))
			for version, bc := range c.Releases.BreakingChanges {
				m.BreakingChanges[version] = pipelineservice.BreakingChange{
					Message:                   bc.Message,
					MigrationDocumentationURL: bc.MigrationDocumentationURL,
					UpgradeDeadline:           bc.UpgradeDeadline,
				}
			}
		}
		m.MigrationDocumentationURL = c.Releases.MigrationDocumentationURL
	}

	// Resource requirements.
	if c.ResourceRequirements != nil && len(c.ResourceRequirements.JobSpecific) > 0 {
		rr := &pipelineservice.ResourceRequirements{}
		for _, js := range c.ResourceRequirements.JobSpecific {
			rr.JobSpecific = append(rr.JobSpecific, pipelineservice.JobSpecificResourceRequirement{
				JobType: js.JobType,
				ResourceRequirements: pipelineservice.ResourceRequirementValues{
					MemoryLimit:   js.ResourceRequirements.MemoryLimit,
					MemoryRequest: js.ResourceRequirements.MemoryRequest,
					CPULimit:      js.ResourceRequirements.CPULimit,
					CPURequest:    js.ResourceRequirements.CPURequest,
				},
			})
		}
		m.ResourceRequirements = rr
	}

	// External documentation URLs.
	for _, doc := range c.ExternalDocumentationURLs {
		m.ExternalDocumentationURLs = append(m.ExternalDocumentationURLs, pipelineservice.ExternalDocumentationURL{
			Title: doc.Title,
			Type:  doc.Type,
			URL:   doc.URL,
		})
	}

	// Suggested streams.
	if c.SuggestedStreams != nil {
		m.SuggestedStreams = c.SuggestedStreams.Streams
	}

	// Allowed hosts.
	if c.AllowedHosts != nil {
		m.AllowedHosts = c.AllowedHosts.Hosts
	}

	return m
}

// marshalConnectorSpec wraps connectionSpecification in a ConnectorSpecification-compatible
// JSON object, matching the format returned by Docker-based connector spec discovery.
func marshalConnectorSpec(spec *airbyteSpec, documentationURL string) string {
	if spec == nil || len(spec.ConnectionSpecification) == 0 {
		return ""
	}
	wrapper := struct {
		DocumentationURL        string          `json:"documentationUrl,omitempty"`
		ConnectionSpecification json.RawMessage `json:"connectionSpecification"`
	}{
		DocumentationURL:        documentationURL,
		ConnectionSpecification: spec.ConnectionSpecification,
	}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return ""
	}
	return string(data)
}
