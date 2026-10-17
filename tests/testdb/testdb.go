// Package testdb prepares the Postgres database that database tests share.
package testdb

import (
	"database/sql"
	"io/fs"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/stretchr/testify/require"
)

// Migrate applies the migrations in fsys, recording them in table. It holds a
// Postgres session lock while it runs, so test packages that run in parallel
// against one database apply migrations one at a time.
func Migrate(t *testing.T, sqlDB *sql.DB, fsys fs.FS, table string) {
	t.Helper()

	locker, err := lock.NewPostgresSessionLocker()
	require.NoError(t, err)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys,
		goose.WithTableName(table),
		goose.WithSessionLocker(locker),
	)
	require.NoError(t, err)

	_, err = provider.Up(t.Context())
	require.NoError(t, err)
}
