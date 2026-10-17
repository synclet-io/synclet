package pipelineexecdocker

import (
	"errors"
	"fmt"
)

// OwnerLabel marks every container with the id of the Synclet instance that created it.
const OwnerLabel = "synclet.io/owner"

// ResolveOwnerID returns the configured owner id, or the hostname when none is configured.
func ResolveOwnerID(configured string, hostname func() (string, error)) (string, error) {
	if configured != "" {
		return configured, nil
	}

	name, err := hostname()
	if err != nil {
		return "", fmt.Errorf("reading hostname for the default owner id, set DOCKER_EXECUTOR_OWNER_ID: %w", err)
	}

	if name == "" {
		return "", errors.New("hostname is empty, set DOCKER_EXECUTOR_OWNER_ID")
	}

	return name, nil
}
