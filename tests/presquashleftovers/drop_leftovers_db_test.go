// Package presquashleftovers_test runs the migrations that drop the objects
// installs built before the history squash still carry, against scratch
// databases. It needs TEST_DATABASE_DSN with a role that may create databases.
package presquashleftovers_test

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelinedbstate"
	"github.com/synclet-io/synclet/modules/workspace/workspacedbstate"
)

const (
	pipelineBeforeDrop  = 20261020000002
	workspaceBeforeDrop = 20261016000001
)

type module struct {
	fsys   fs.FS
	table  string
	before int64
}

func modules() []module {
	return []module{
		{fsys: pipelinedbstate.MigrationsFS, table: "pipeline_goose_db_version", before: pipelineBeforeDrop},
		{fsys: workspacedbstate.MigrationsFS, table: "workspace_goose_db_version", before: workspaceBeforeDrop},
	}
}

func TestMigrationsDropPreSquashLeftovers(t *testing.T) {
	db := scratchDatabase(t)

	for _, m := range modules() {
		_, err := provider(t, db, m).UpTo(t.Context(), m.before)
		require.NoError(t, err)
	}

	exec(t, db, `CREATE TABLE pipeline.schema_changes (
		id uuid DEFAULT gen_random_uuid() NOT NULL,
		connection_id uuid NOT NULL,
		changes_json jsonb NOT NULL,
		detected_at timestamp with time zone DEFAULT now() NOT NULL,
		CONSTRAINT schema_changes_pkey PRIMARY KEY (id)
	)`)
	exec(t, db, `CREATE INDEX idx_schema_changes_connection_id ON pipeline.schema_changes USING btree (connection_id)`)
	exec(t, db, `ALTER TABLE workspace.workspaces ADD COLUMN max_jobs_per_workspace integer DEFAULT 0 NOT NULL`)
	exec(t, db, `INSERT INTO workspace.workspaces (name, slug, max_jobs_per_workspace) VALUES ('w', 'w', 5)`)

	for _, m := range modules() {
		_, err := provider(t, db, m).Up(t.Context())
		require.NoError(t, err)
	}

	requireLeftoversGone(t, db)

	var workspaces int
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT count(*) FROM workspace.workspaces`).Scan(&workspaces))
	require.Equal(t, 1, workspaces)
}

func TestMigrationsDropPreSquashLeftoversOnFreshDatabase(t *testing.T) {
	db := scratchDatabase(t)

	for _, m := range modules() {
		p := provider(t, db, m)

		_, err := p.Up(t.Context())
		require.NoError(t, err)

		_, err = p.DownTo(t.Context(), m.before)
		require.NoError(t, err)

		_, err = p.Up(t.Context())
		require.NoError(t, err)
	}

	requireLeftoversGone(t, db)
}

func requireLeftoversGone(t *testing.T, db *sql.DB) {
	t.Helper()

	var table, index sql.NullString
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT to_regclass('pipeline.schema_changes')::text, to_regclass('pipeline.idx_schema_changes_connection_id')::text`,
	).Scan(&table, &index))
	require.False(t, table.Valid, "pipeline.schema_changes still exists")
	require.False(t, index.Valid, "pipeline.idx_schema_changes_connection_id still exists")

	var columns int
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT count(*) FROM information_schema.columns
		 WHERE table_schema = 'workspace' AND table_name = 'workspaces' AND column_name = 'max_jobs_per_workspace'`,
	).Scan(&columns))
	require.Zero(t, columns, "workspace.workspaces.max_jobs_per_workspace still exists")
}

func provider(t *testing.T, db *sql.DB, m module) *goose.Provider {
	t.Helper()

	p, err := goose.NewProvider(goose.DialectPostgres, db, m.fsys,
		goose.WithTableName(m.table),
		goose.WithDisableGlobalRegistry(true),
	)
	require.NoError(t, err)

	return p
}

func exec(t *testing.T, db *sql.DB, query string) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), query)
	require.NoError(t, err)
}

// scratchDatabase creates an empty database through TEST_DATABASE_DSN, drops
// it when the test ends, and returns a connection to it.
func scratchDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping pre-squash leftovers migration test")
	}

	admin, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })

	name := fmt.Sprintf("synclet_leftovers_test_%d", time.Now().UnixNano())
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

	db := stdlib.OpenDB(*config)

	t.Cleanup(func() { require.NoError(t, db.Close()) })

	return db
}
