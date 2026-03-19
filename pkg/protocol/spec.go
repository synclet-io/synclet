package protocol

import "encoding/json"

// ConnectorSpecification describes the configuration schema and capabilities of a connector.
type ConnectorSpecification struct {
	DocumentationURL              string                `json:"documentationUrl,omitempty"`
	ChangelogURL                  string                `json:"changelogUrl,omitempty"`
	ConnectionSpecification       json.RawMessage       `json:"connectionSpecification"`
	SupportsIncremental           bool                  `json:"supportsIncremental,omitempty"`
	SupportsNormalization         bool                  `json:"supportsNormalization,omitempty"`
	SupportsDBT                   bool                  `json:"supportsDBT,omitempty"`
	SupportedDestinationSyncModes []DestinationSyncMode `json:"supported_destination_sync_modes,omitempty"`
	AdvancedAuth                  *AdvancedAuth         `json:"advancedAuth,omitempty"`
	ProtocolVersion               string                `json:"protocol_version,omitempty"`
}

// AdvancedAuth describes OAuth and other advanced authentication flows for a connector.
type AdvancedAuth struct {
	AuthFlowType            string                   `json:"auth_flow_type,omitempty"`
	PredicateKey            []string                 `json:"predicate_key,omitempty"`
	PredicateValue          string                   `json:"predicate_value,omitempty"`
	OAuthConfigSpecification *OAuthConfigSpecification `json:"oauth_config_specification,omitempty"`
}

// OAuthConfigSpecification describes the OAuth configuration parameters for a connector.
type OAuthConfigSpecification struct {
	OAuthUserInputFromConnectorConfigSpecification json.RawMessage `json:"oauthUserInputFromConnectorConfigSpecification,omitempty"`
	CompleteOAuthOutputSpecification               json.RawMessage `json:"completeOAuthOutputSpecification,omitempty"`
	CompleteOAuthServerInputSpecification          json.RawMessage `json:"completeOAuthServerInputSpecification,omitempty"`
	CompleteOAuthServerOutputSpecification         json.RawMessage `json:"completeOAuthServerOutputSpecification,omitempty"`
}
