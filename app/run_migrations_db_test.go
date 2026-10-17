package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/migrations"
)

// TestMigrationProvidersSerialiseConcurrentMigrators runs every module's
// migrations from two connection pools at once, as two server replicas
// starting together do, against a database of its own whose version tables
// exist and whose migrations are all pending. It needs TEST_DATABASE_DSN with
// a role that may create databases.
func TestMigrationProvidersSerialiseConcurrentMigrators(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping concurrent migration test")
	}

	config := createScratchDatabase(t, dsn)

	const migrators = 2

	pools := make([]*sql.DB, migrators)

	for i := range pools {
		pools[i] = stdlib.OpenDB(*config)

		t.Cleanup(func() { require.NoError(t, pools[i].Close()) })
	}

	for _, source := range migrations.Sources() {
		provider, err := newMigrationProvider(pools[0], source)
		require.NoError(t, err)

		version, err := provider.GetDBVersion(t.Context())
		require.NoError(t, err)
		require.Zero(t, version, "module %s", source.Module)
	}

	var wg sync.WaitGroup

	start := make(chan struct{})
	errs := make([]error, migrators)

	for i, sqlDB := range pools {
		wg.Go(func() {
			<-start

			for _, source := range migrations.Sources() {
				provider, err := newMigrationProvider(sqlDB, source)
				if err != nil {
					errs[i] = err

					return
				}

				if _, err := provider.Up(t.Context()); err != nil {
					errs[i] = err

					return
				}
			}
		})
	}

	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "migrator %d", i)
	}
}

// createScratchDatabase creates an empty database through dsn, drops it when
// the test ends, and returns dsn's connection config pointed at it.
func createScratchDatabase(t *testing.T, dsn string) *pgx.ConnConfig {
	t.Helper()

	admin, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	name := fmt.Sprintf("synclet_migrate_test_%d", time.Now().UnixNano())
	ident := pgx.Identifier{name}.Sanitize()

	_, err = admin.ExecContext(t.Context(), "CREATE DATABASE "+ident)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := admin.ExecContext(context.WithoutCancel(t.Context()), "DROP DATABASE "+ident+" WITH (FORCE)")
		require.NoError(t, err)
	})

	config, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)

	config.Database = name

	return config
}
