package workspacedbstate

import (
	"embed"

	"github.com/synclet-io/synclet/pkg/migrations"
)

//go:embed *.sql
var MigrationsFS embed.FS

func init() {
	migrations.Register("workspace", MigrationsFS)
}
