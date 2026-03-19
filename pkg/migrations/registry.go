package migrations

import (
	"embed"
)

// Source represents a module's migration files.
type Source struct {
	Module string
	FS     embed.FS
}

// sources is the ordered list of module migrations.
//
//nolint:gochecknoglobals
var sources []Source

// Register adds a module's migration source.
func Register(module string, fs embed.FS) {
	sources = append(sources, Source{Module: module, FS: fs})
}

// Sources returns all registered migration sources.
func Sources() []Source {
	return sources
}
