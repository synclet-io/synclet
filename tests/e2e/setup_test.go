//go:build e2e

package e2e

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	// Import all module dbstate packages so their init() functions register migrations.
	_ "github.com/synclet-io/synclet/modules/auth/authdbstate"
	_ "github.com/synclet-io/synclet/modules/notify/notifydbstate"
	_ "github.com/synclet-io/synclet/modules/pipeline/pipelinedbstate"
	_ "github.com/synclet-io/synclet/modules/workspace/workspacedbstate"

	"github.com/synclet-io/synclet/pkg/migrations"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql
)

// syncletBinary is the path to the built synclet binary used by all tests.
var syncletBinary string

// testDB is the shared database connection for migrations and table truncation.
var testDB *sql.DB

// testDSN is the PostgreSQL DSN used by both setup and server subprocesses.
var testDSN string

// kindClusterName is set when E2E_K8S=1 and kind cluster is created.
var kindClusterName string

// kindKubeconfig is the path to the kind cluster kubeconfig.
var kindKubeconfig string

func TestMain(m *testing.M) {
	// Determine project root (two levels up from tests/e2e/).
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	// Step 1: Build synclet binary.
	log.Println("Building synclet binary...")
	syncletBinary = filepath.Join(projectRoot, "bin", "synclet-test")
	buildCmd := exec.Command("go", "build", "-o", syncletBinary, ".")
	buildCmd.Dir = projectRoot
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Failed to build synclet binary: %v", err)
	}
	log.Printf("Built synclet binary: %s", syncletBinary)

	// Step 2: Build test connector Docker images locally.
	log.Println("Building test connector Docker images...")
	for _, img := range []struct {
		name       string
		dockerfile string
	}{
		{testSourceImage, "tests/e2e/testconnector/source/Dockerfile"},
		{testDestImage, "tests/e2e/testconnector/destination/Dockerfile"},
	} {
		cmd := exec.Command("docker", "build", "-t", img.name, "-f", img.dockerfile, "tests/e2e/testconnector/")
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to build image %s: %v", img.name, err)
		}
	}
	log.Println("Test connector images built")

	// Step 3: Connect to PostgreSQL and run migrations.
	testDSN = os.Getenv("E2E_DATABASE_URL")
	if testDSN == "" {
		testDSN = "postgres://postgres:postgres@localhost:5432/synclet_test?sslmode=disable"
	}

	var err error
	testDB, err = sql.Open("pgx", testDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer testDB.Close()

	if err := testDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Running migrations...")
	for _, source := range migrations.Sources() {
		goose.SetTableName(source.Module + "_goose_db_version")
		goose.SetBaseFS(source.FS)
		if err := goose.Up(testDB, "."); err != nil {
			log.Fatalf("Failed to run %s migrations: %v", source.Module, err)
		}
	}
	log.Println("Migrations complete")

	// Step 4: Create kind cluster if E2E_K8S=1.
	if os.Getenv("E2E_K8S") == "1" {
		setupKindCluster()
	}

	// Step 5: Run tests.
	code := m.Run()

	// Step 6: Teardown.
	if kindClusterName != "" {
		teardownKindCluster()
	}

	os.Exit(code)
}

func setupKindCluster() {
	kindClusterName = fmt.Sprintf("synclet-e2e-%d", time.Now().UnixNano())
	log.Printf("Creating kind cluster: %s", kindClusterName)

	cmd := exec.Command("kind", "create", "cluster", "--name", kindClusterName, "--wait", "3m")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to create kind cluster: %v", err)
	}

	// Export kubeconfig.
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		log.Fatalf("Failed to create temp kubeconfig: %v", err)
	}
	tmpFile.Close()

	kindKubeconfig = tmpFile.Name()
	cmd = exec.Command("kind", "get", "kubeconfig", "--name", kindClusterName)
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to get kubeconfig: %v", err)
	}
	if err := os.WriteFile(kindKubeconfig, out, 0o600); err != nil {
		log.Fatalf("Failed to write kubeconfig: %v", err)
	}

	// Load test images into kind cluster.
	for _, img := range []string{testSourceImage, testDestImage} {
		cmd := exec.Command("kind", "load", "docker-image", img, "--name", kindClusterName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Warning: failed to load image %s into kind: %v", img, err)
		}
	}

	log.Printf("Kind cluster %s ready, kubeconfig: %s", kindClusterName, kindKubeconfig)
}

func teardownKindCluster() {
	log.Printf("Deleting kind cluster: %s", kindClusterName)
	cmd := exec.Command("kind", "delete", "cluster", "--name", kindClusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to delete kind cluster: %v", err)
	}
	if kindKubeconfig != "" {
		os.Remove(kindKubeconfig)
	}
}

// seedTestConnectors inserts repository and connector records for test images.
func seedTestConnectors(t *testing.T) {
	t.Helper()
	if testDB == nil {
		return
	}

	// Check if already seeded.
	var count int
	err := testDB.QueryRow("SELECT COUNT(*) FROM pipeline.repository_connectors WHERE docker_repository = $1", testSourceImage).Scan(&count)
	if err == nil && count > 0 {
		return // Already seeded.
	}

	// Create a test repository (workspace_id is a dummy UUID for e2e).
	repoID := "00000000-0000-0000-0000-000000000001"
	wsID := "00000000-0000-0000-0000-000000000099"
	_, err = testDB.Exec(`INSERT INTO pipeline.repositories (id, workspace_id, name, url, status, last_synced_at, created_at, updated_at)
		VALUES ($1, $2, 'e2e-test-repo', 'local://test', 'synced', NOW(), NOW(), NOW()) ON CONFLICT DO NOTHING`, repoID, wsID)
	if err != nil {
		t.Logf("Warning: could not seed repository: %v", err)
	}

	// Insert test source connector.
	_, err = testDB.Exec(`INSERT INTO pipeline.repository_connectors (id, repository_id, name, docker_repository, docker_image_tag, connector_type)
		VALUES (gen_random_uuid(), $1, 'Test Source', $2, 'latest', 'source') ON CONFLICT DO NOTHING`,
		repoID, testSourceImage)
	if err != nil {
		t.Logf("Warning: could not seed source connector: %v", err)
	}

	// Insert test destination connector.
	_, err = testDB.Exec(`INSERT INTO pipeline.repository_connectors (id, repository_id, name, docker_repository, docker_image_tag, connector_type)
		VALUES (gen_random_uuid(), $1, 'Test Destination', $2, 'latest', 'destination') ON CONFLICT DO NOTHING`,
		repoID, testDestImage)
	if err != nil {
		t.Logf("Warning: could not seed destination connector: %v", err)
	}
}

// truncateTestTables clears test-related tables for per-test isolation.
func truncateTestTables(t *testing.T) {
	t.Helper()
	if testDB == nil {
		return
	}
	tables := []string{
		"pipeline.job_attempts",
		"pipeline.jobs",
		"pipeline.connection_state",
		"pipeline.configured_catalogs",
		"pipeline.catalog_discoveries",
		"pipeline.connections",
		"pipeline.sources",
		"pipeline.destinations",
		"pipeline.secrets",
		"workspace.workspace_members",
		"workspace.workspace_invites",
		"workspace.workspaces",
		"auth.refresh_tokens",
		"auth.api_keys",
		"auth.oidc_identities",
		"auth.users",
	}
	for _, table := range tables {
		if _, err := testDB.Exec("TRUNCATE TABLE " + table + " CASCADE"); err != nil {
			// Table might not exist in test DB, that's OK.
			t.Logf("Warning: could not truncate %s: %v", table, err)
		}
	}
}
