// Package buildinfo reports the version of the running synclet binary.
package buildinfo

import "runtime/debug"

// version is set at build time:
//
//	go build -ldflags "-X github.com/synclet-io/synclet/pkg/buildinfo.version=v0.4.0"
//
//nolint:gochecknoglobals
var version = ""

// Version returns the -ldflags version, else the VCS revision Go stamped
// into the binary, else "dev".
func Version() string {
	if version != "" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 12 {
			return "dev-" + setting.Value[:12]
		}
	}

	return "dev"
}
