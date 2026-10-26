package pipelineexec

import (
	"fmt"
	"strings"
)

// OwnerLabel marks every container and Job with the id of the executor that created it.
const OwnerLabel = "synclet.io/owner"

// ResolveOwnerID returns the executor's owner id: id when it is set, or
// "<prefix>-<n>" when prefix is set, n being the digits after the last "-" of
// the hostname, which for a StatefulSet pod is its ordinal. setting and
// prefixSetting name the variables that configure id and prefix, and sharedBy
// what executors that need distinct ids share, such as "Docker daemon", for
// the error; an executor without a prefix variable passes an empty prefix.
func ResolveOwnerID(id, prefix, setting, prefixSetting, sharedBy string, hostname func() (string, error)) (string, error) {
	switch {
	case id != "" && prefix != "":
		return "", fmt.Errorf("%s and %s are both set: set only one", setting, prefixSetting)
	case id != "":
		return id, nil
	case prefix == "":
		return "", fmt.Errorf("%s is not set: set it to an id unique to this executor among executors sharing its %s", setting, sharedBy)
	}

	name, err := hostname()
	if err != nil {
		return "", fmt.Errorf("reading the hostname for the ordinal of %s: %w", prefixSetting, err)
	}

	cut := strings.LastIndex(name, "-")
	ordinal := name[cut+1:]

	if cut < 0 || ordinal == "" || strings.Trim(ordinal, "0123456789") != "" {
		return "", fmt.Errorf("%s is set but hostname %q does not end in -<ordinal>: set %s instead", prefixSetting, name, setting)
	}

	return prefix + "-" + ordinal, nil
}
