package pipelineconnectors

import (
	"sort"

	"github.com/Masterminds/semver/v3"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// VersionedBreakingChange extends the metadata BreakingChange with the version it applies to.
type VersionedBreakingChange struct {
	Version                   string
	Message                   string
	MigrationDocumentationURL string
	UpgradeDeadline           string
}

// DetectBreakingChanges returns breaking changes between currentTag (exclusive)
// and targetTag (inclusive), sorted by semver ascending.
// Returns nil if either tag is not valid semver, metadata is nil, or no changes found.
func DetectBreakingChanges(currentTag, targetTag string, metadata *pipelineservice.RepositoryConnectorMetadata) []VersionedBreakingChange {
	if metadata == nil || len(metadata.BreakingChanges) == 0 {
		return nil
	}

	currentVer, err := semver.StrictNewVersion(currentTag)
	if err != nil {
		return nil
	}

	targetVer, err := semver.StrictNewVersion(targetTag)
	if err != nil {
		return nil
	}

	if !currentVer.LessThan(targetVer) {
		return nil
	}

	type versionedEntry struct {
		ver   *semver.Version
		entry VersionedBreakingChange
	}

	var collected []versionedEntry

	for versionStr, change := range metadata.BreakingChanges {
		ver, parseErr := semver.StrictNewVersion(versionStr)
		if parseErr != nil {
			continue
		}

		// current < breakingVersion <= target
		if currentVer.LessThan(ver) && (ver.LessThan(targetVer) || ver.Equal(targetVer)) {
			collected = append(collected, versionedEntry{
				ver: ver,
				entry: VersionedBreakingChange{
					Version:                   versionStr,
					Message:                   change.Message,
					MigrationDocumentationURL: change.MigrationDocumentationURL,
					UpgradeDeadline:           change.UpgradeDeadline,
				},
			})
		}
	}

	if len(collected) == 0 {
		return nil
	}

	sort.Slice(collected, func(i, j int) bool {
		return collected[i].ver.LessThan(collected[j].ver)
	})

	result := make([]VersionedBreakingChange, len(collected))
	for i, c := range collected {
		result[i] = c.entry
	}

	return result
}
