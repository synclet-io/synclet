package appmcp

import (
	"os"
	"strconv"
)

// Config switches the MCP server and its write tools.
type Config struct {
	// Enabled starts the MCP HTTP server.
	Enabled bool `env:"ENABLED" envDefault:"false"`
	// WriteTools lists the tools that create, edit and delete entities.
	WriteTools bool `env:"WRITE_TOOLS" envDefault:"true"`
	// PublicURL is the base URL clients reach the MCP server at: the OAuth
	// protected resource and authorization server issuer.
	PublicURL string `env:"PUBLIC_URL" envDefault:"http://localhost:8082"`
	// UIURL is the base URL of the main UI, which serves the OAuth consent
	// page. Empty takes AUTH_OIDC_CALLBACK_BASE_URL.
	UIURL string `env:"UI_URL"`
}

// ConfigPrefix is the environment prefix of Config.
const ConfigPrefix = "MCP_"

// EnabledFromEnv reports whether MCP_ENABLED turns the MCP server on. It is
// read before the dependency graph is built, since it decides whether the
// graph holds the server at all.
func EnabledFromEnv() bool {
	enabled, err := strconv.ParseBool(os.Getenv(ConfigPrefix + "ENABLED"))

	return err == nil && enabled
}
