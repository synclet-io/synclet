package authservice

// OIDCProviderInfo is a lightweight provider descriptor for the frontend.
type OIDCProviderInfo struct {
	Slug        string
	DisplayName string
}

// GetOIDCProviders returns the list of configured OIDC provider slugs and display names.
type GetOIDCProviders struct {
	providers map[string]*OIDCProvider
}

// NewGetOIDCProviders creates a new GetOIDCProviders use case.
func NewGetOIDCProviders(providers map[string]*OIDCProvider) *GetOIDCProviders {
	return &GetOIDCProviders{providers: providers}
}

// Execute returns the list of configured OIDC providers.
func (uc *GetOIDCProviders) Execute() []OIDCProviderInfo {
	result := make([]OIDCProviderInfo, 0, len(uc.providers))
	for _, p := range uc.providers {
		result = append(result, OIDCProviderInfo{
			Slug:        p.Config.Slug,
			DisplayName: p.Config.DisplayName,
		})
	}

	return result
}
