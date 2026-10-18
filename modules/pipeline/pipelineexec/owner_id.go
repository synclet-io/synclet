package pipelineexec

import (
	"fmt"
)

// OwnerLabel marks every container and Job with the id of the executor that created it.
const OwnerLabel = "synclet.io/owner"

// ResolveOwnerID returns the configured owner id, or the hostname when none is
// configured. setting names the variable that configures it, for the error.
func ResolveOwnerID(configured, setting string, hostname func() (string, error)) (string, error) {
	if configured != "" {
		return configured, nil
	}

	name, err := hostname()
	if err != nil {
		return "", fmt.Errorf("reading hostname for the default owner id, set %s: %w", setting, err)
	}

	if name == "" {
		return "", fmt.Errorf("hostname is empty, set %s", setting)
	}

	return name, nil
}
