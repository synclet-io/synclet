//go:build e2e

package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
)

// TestTestedWith runs real connector images end to end and produces the
// "Tested with" matrix of the README and the docs. It pulls large images and
// is skipped unless TESTED_WITH=1. Connector containers reach the test
// PostgreSQL through TESTED_WITH_PG_HOST (default host.docker.internal) and
// the port of E2E_DATABASE_URL.
func TestTestedWith(t *testing.T) {
	if os.Getenv("TESTED_WITH") != "1" {
		t.Skip("set TESTED_WITH=1 to run real connector images")
	}

	dsn, err := url.Parse(testDSN)
	require.NoError(t, err)

	pg := twPostgres{
		host:     envOr("TESTED_WITH_PG_HOST", "host.docker.internal"),
		port:     dsn.Port(),
		user:     dsn.User.Username(),
		password: func() string { p, _ := dsn.User.Password(); return p }(),
		dsn:      dsn,
	}
	if pg.port == "" {
		pg.port = "5432"
	}

	pg.recreateDatabase(t, "tw_source")
	src := pg.open(t, "tw_source")
	_, err = src.Exec(`CREATE TABLE public.cities (id int PRIMARY KEY, name text, updated_at timestamptz DEFAULT now());
		INSERT INTO public.cities (id, name) SELECT g, 'city ' || g FROM generate_series(1, 250) g`)
	require.NoError(t, err, "seeding tw_source")
	require.NoError(t, src.Close())

	faker := twImage{"airbyte/source-faker", "6.2.38"}
	sourcePostgres := twImage{"airbyte/source-postgres", "3.8.7"}
	goSourcePostgres := twImage{"ghcr.io/synclet-io/connectors/source-postgres", "0.2.1"}
	destinationPostgres := twImage{"airbyte/destination-postgres", "3.0.23"}
	destinationDevNull := twImage{"airbyte/destination-dev-null", "0.7.24"}
	destinationDuckDB := twImage{"airbyte/destination-duckdb", "0.6.0"}
	fakerConfig := map[string]any{"count": 100, "seed": 1, "records_per_slice": 1000, "always_updated": false, "parallelism": 1}

	cases := []twCase{
		{
			name: "faker-incremental-append", source: faker, destination: destinationPostgres,
			sourceConfig: fakerConfig, targetDB: "tw_target_faker_inc", table: "x.users",
			streams: []streamOpt{{Name: "users", SyncMode: "incremental", CursorField: "updated_at", DestinationSyncMode: "append"}},
			rows:    []int{100, 100},
		},
		{
			name: "source-postgres-incremental-append", source: sourcePostgres, destination: destinationPostgres,
			sourceConfig: pg.sourceConfig("tw_source"), targetDB: "tw_target_postgres_inc", table: "public.cities",
			streams: []streamOpt{{Name: "cities", Namespace: "public", SyncMode: "incremental", CursorField: "id", DestinationSyncMode: "append"}},
			rows:    []int{250, 251},
		},
		{
			// The Go source reads cursor >= saved, re-reading the cursor row.
			name: "go-source-postgres-incremental-append", source: goSourcePostgres, destination: destinationPostgres,
			sourceConfig: pg.sourceConfig("tw_source"), targetDB: "tw_target_go_postgres_inc", table: "public.cities",
			streams: []streamOpt{{Name: "cities", Namespace: "public", SyncMode: "incremental", CursorField: "id", DestinationSyncMode: "append"}},
			rows:    []int{250, 251},
		},
		{
			name: "faker-full-refresh-overwrite", source: faker, destination: destinationPostgres,
			sourceConfig: fakerConfig, targetDB: "tw_target_faker_fr", table: "x.users",
			streams: []streamOpt{{Name: "users", SyncMode: "full_refresh", DestinationSyncMode: "overwrite"}},
			rows:    []int{100, 100},
		},
		{
			// The DuckDB file outlives the job in the executor's local root.
			name: "faker-incremental-append-duckdb", source: faker, destination: destinationDuckDB,
			sourceConfig:      fakerConfig,
			destinationConfig: map[string]any{"destination_path": "/local/destination.duckdb", "schema": "x"},
			streams:           []streamOpt{{Name: "users", SyncMode: "incremental", CursorField: "updated_at", DestinationSyncMode: "append"}},
			recordsRead:       []int64{100, 0},
			localFile:         "destination.duckdb",
		},
		{
			name: "faker-dev-null", source: faker, destination: destinationDevNull,
			sourceConfig:      fakerConfig,
			destinationConfig: map[string]any{"test_destination": map[string]any{"test_destination_type": "SILENT"}},
			streams:           []streamOpt{{Name: "users", SyncMode: "full_refresh", DestinationSyncMode: "overwrite"}},
			recordsRead:       []int64{100, 100},
		},
	}

	// Synclet creates the root itself, with mode 1777, so a connector of any
	// uid writes to it.
	localRoot := filepath.Join(t.TempDir(), "local")

	server := startSyncletServer(t, map[string]string{"DOCKER_EXECUTOR_LOCAL_ROOT": localRoot})
	defer server.Stop()

	client := newAPIClient(server.baseURL)
	client.RegisterAndLogin(context.Background(), t)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			destConfig := tc.destinationConfig
			if destConfig == nil {
				pg.recreateDatabase(t, tc.targetDB)
				destConfig = pg.destinationConfig(tc.targetDB, "x")
			}

			connID := tc.createConnection(ctx, t, client, destConfig)

			for run := range max(len(tc.rows), len(tc.recordsRead)) {
				var localPath string

				var before os.FileInfo

				if tc.localFile != "" {
					localPath = filepath.Join(localRoot, tc.localFile)
					before, _ = os.Stat(localPath)
				}

				res := client.WaitForJob(ctx, t, client.TriggerSync(ctx, t, connID), 10*time.Minute)
				require.NoError(t, res.Err, "run %d", run+1)

				if run < len(tc.recordsRead) {
					t.Logf("TESTED-WITH %s -> %s | %s | run %d: status=%s records_read=%d",
						tc.source, tc.destination, tc.name, run+1, res.JobStatus, res.RecordsRead)
					assert.Equal(t, tc.recordsRead[run], res.RecordsRead, "records read in run %d", run+1)

					if localPath != "" {
						rootInfo, err := os.Stat(localRoot)
						require.NoError(t, err, "local root after run %d", run+1)
						t.Logf("TESTED-WITH %s | run %d: local root %s", tc.name, run+1, rootInfo.Mode())

						after, err := os.Stat(localPath)
						require.NoError(t, err, "%s on the host after run %d", tc.localFile, run+1)
						t.Logf("TESTED-WITH %s | run %d: %s left on the host, %d bytes, modified %s",
							tc.name, run+1, localPath, after.Size(), after.ModTime().Format(time.RFC3339Nano))

						if tc.recordsRead[run] > 0 {
							written := before == nil || after.Size() != before.Size() || after.ModTime().After(before.ModTime())
							assert.True(t, written, "run %d read %d records but did not write %s", run+1, tc.recordsRead[run], localPath)
						}
					}

					continue
				}

				target := pg.open(t, tc.targetDB)
				var rows int
				require.NoError(t, target.QueryRow("SELECT count(*) FROM "+tc.table).Scan(&rows))
				require.NoError(t, target.Close())

				t.Logf("TESTED-WITH %s -> %s | %s | run %d: status=%s records_read=%d rows=%d",
					tc.source, tc.destination, tc.name, run+1, res.JobStatus, res.RecordsRead, rows)

				assert.Equal(t, tc.rows[run], rows, "rows in %s after run %d", tc.table, run+1)
			}
		})
	}
}

type twImage struct{ repo, tag string }

func (i twImage) String() string { return i.repo + ":" + i.tag }

type twCase struct {
	name                string
	source, destination twImage
	sourceConfig        map[string]any
	// destinationConfig configures a destination other than the test
	// PostgreSQL; nil writes to targetDB there.
	destinationConfig map[string]any
	targetDB, table   string
	streams           []streamOpt
	// rows is the expected row count in table after each run.
	rows []int
	// recordsRead is the expected records read by each run, for a destination
	// whose output the test cannot query.
	recordsRead []int64
	// localFile is a file the destination leaves in the executor's local
	// root, checked after each run.
	localFile string
}

func (tc twCase) createConnection(ctx context.Context, t *testing.T, c *apiClient, destConfig map[string]any) string {
	t.Helper()

	src, err := c.source.CreateSource(ctx, newAuthRequest(c, &pipelinev1.CreateSourceRequest{
		WorkspaceId:        c.workspaceID,
		Name:               tc.name + "-src",
		ManagedConnectorId: c.imageConnector(t, tc.source, "source"),
		Config:             mapToStruct(t, tc.sourceConfig),
	}))
	require.NoError(t, err, "creating source")

	dst, err := c.destination.CreateDestination(ctx, newAuthRequest(c, &pipelinev1.CreateDestinationRequest{
		WorkspaceId:        c.workspaceID,
		Name:               tc.name + "-dst",
		ManagedConnectorId: c.imageConnector(t, tc.destination, "destination"),
		Config:             mapToStruct(t, destConfig),
	}))
	require.NoError(t, err, "creating destination")

	sourceID := src.Msg.GetSource().GetId()
	connID := c.CreateConnection(ctx, t, tc.name, sourceID, dst.Msg.GetDestination().GetId())
	c.DiscoverSourceSchema(ctx, t, sourceID)
	c.ConfigureStreams(ctx, t, connID, tc.streams)

	return connID
}

// imageConnector adds a pinned connector image to the client's workspace and returns its ID.
func (c *apiClient) imageConnector(t *testing.T, img twImage, connectorType string) string {
	t.Helper()

	spec, err := readConnectorSpec(img.String())
	require.NoError(t, err, "reading spec of %s", img)

	var id string
	err = testDB.QueryRow(`INSERT INTO pipeline.managed_connectors (id, workspace_id, docker_image, docker_tag, name, connector_type, spec)
		VALUES (gen_random_uuid(), $1, $2, $3, $2, $4, $5) RETURNING id`, c.workspaceID, img.repo, img.tag, connectorType, spec).Scan(&id)
	require.NoError(t, err, "seeding managed connector %s", img)

	return id
}

type twPostgres struct {
	host, port, user, password string
	dsn                        *url.URL
}

func (p twPostgres) open(t *testing.T, database string) *sql.DB {
	t.Helper()

	u := *p.dsn
	u.Path = "/" + database
	db, err := sql.Open("pgx", u.String())
	require.NoError(t, err)

	return db
}

func (p twPostgres) recreateDatabase(t *testing.T, name string) {
	t.Helper()

	_, err := testDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, name))
	require.NoError(t, err)
	_, err = testDB.Exec(fmt.Sprintf(`CREATE DATABASE %q`, name))
	require.NoError(t, err)
}

func (p twPostgres) sourceConfig(database string) map[string]any {
	return map[string]any{
		"host": p.host, "port": p.portNumber(), "database": database, "schemas": []any{"public"},
		"username": p.user, "password": p.password,
		"ssl_mode":           map[string]any{"mode": "disable"},
		"replication_method": map[string]any{"method": "Standard"},
		"tunnel_method":      map[string]any{"tunnel_method": "NO_TUNNEL"},
	}
}

func (p twPostgres) destinationConfig(database, schema string) map[string]any {
	return map[string]any{
		"host": p.host, "port": p.portNumber(), "database": database, "schema": schema,
		"username": p.user, "password": p.password,
		"ssl_mode":      map[string]any{"mode": "disable"},
		"tunnel_method": map[string]any{"tunnel_method": "NO_TUNNEL"},
	}
}

func (p twPostgres) portNumber() int {
	n, _ := strconv.Atoi(p.port)

	return n
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
