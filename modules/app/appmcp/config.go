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
