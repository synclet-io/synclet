package pipelineexec

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const modulePathPrefix = "github.com/synclet-io/synclet/modules/"

// allowedModulePaths are the only module packages an executor package may
// import: its own tree, the domain types and the pipeline metrics. A path
// ending in "/" allows the tree below it.
var allowedModulePaths = []string{
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec",
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/",
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice",
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics",
}

// forbiddenDependencies open database connections. A path ending in "/"
// forbids the tree below it.
var forbiddenDependencies = []string{
	"github.com/go-pnp/go-pnp/sql/pnpgorm",
	"gorm.io/driver/",
	"github.com/jackc/pgx/v5",
	"github.com/jackc/pgx/v5/stdlib",
	"github.com/jackc/pgx/v5/pgxpool",
}

// toleratedDependencies reach the closure only through
// pipelineservice/gen.storages.go, which boilerplate-go generates with
// imports of its dbutil (lib/pq for pq.Array), idempotency (pgconn error
// codes) and txoutbox (gorm.io/gorm) packages. The tolerance is transitive
// only: a direct import from any other package fails the test. The
// follow-up is a boilerplate-go release whose generated storages import
// none of the three; this test fails once an entry has left the closure, so
// the entry is deleted then. pgconn matters most: unlike lib/pq it can open
// a connection by itself.
var toleratedDependencies = map[string]string{
	"github.com/lib/pq":              "boilerplate-go lib/dbutil",
	"github.com/jackc/pgx/v5/pgconn": "boilerplate-go lib/idempotency",
	"gorm.io/gorm":                   "boilerplate-go lib/txoutbox",
}

// toleratedImporters are the only packages that may import a tolerated
// dependency directly: pipelineservice, for its generated storages, and
// boilerplate-go itself.
var toleratedImporters = []string{
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice",
	"github.com/saturn4er/boilerplate-go/",
}

func matchesAny(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(path, pattern) || path == pattern {
			return true
		}
	}

	return false
}

// TestExecutorDependencies fails when the executor packages (this package,
// its subpackages and the executor API client) reach storage, use cases or a
// package that opens database connections, or when any package but the
// tolerated importers imports a tolerated or forbidden package directly: an
// executor talks to the control plane through ExecutorBackend alone.
func TestExecutorDependencies(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "list", "-deps", "-f", `{{.ImportPath}} {{join .Imports " "}}`, "./...").Output()
	require.NoError(t, err)

	seen := map[string]bool{}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		dep, imports := fields[0], fields[1:]
		seen[dep] = true

		switch {
		case strings.HasPrefix(dep, modulePathPrefix) && !matchesAny(dep, allowedModulePaths):
			t.Errorf("executor packages reach %s: storage and use cases belong to the control plane", dep)
		case matchesAny(dep, forbiddenDependencies):
			t.Errorf("executor packages reach %s, which opens database connections", dep)
		}

		if matchesAny(dep, toleratedImporters) {
			continue
		}

		for _, imported := range imports {
			if matchesAny(imported, forbiddenDependencies) || toleratedDependencies[imported] != "" {
				t.Errorf("%s imports %s directly: only pipelineservice's generated storages and boilerplate-go may", dep, imported)
			}
		}
	}

	for dep, source := range toleratedDependencies {
		if !seen[dep] {
			t.Errorf("%s (via %s) has left the closure: delete its tolerated entry", dep, source)
		}
	}
}
