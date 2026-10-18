// Package executorregistry_test runs the executor registry upsert against a
// real Postgres. It needs TEST_DATABASE_DSN.
package executorregistry_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/synclet-io/synclet/modules/pipeline/pipelinedbstate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/tests/testdb"
)

type noopLogDelegate struct{}

func (noopLogDelegate) Info(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopLogDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopLogDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopLogDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopLogDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopLogDelegate) Named(string) logging.Delegate                      { return d }
func (d noopLogDelegate) SkipCallers(int) logging.Delegate                   { return d }

func setup(t *testing.T) pipelineservice.ExecutorsStorage {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping executor registry database test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, pipelinedbstate.MigrationsFS, "pipeline_goose_db_version")

	return pipelinestorage.NewExecutorsStorage(db, &logging.Logger{Delegate: noopLogDelegate{}})
}

// TestExecutorsUpsert asserts that a second heartbeat of an executor keeps
// its row id and first_seen_at and refreshes every reported column and
// last_seen_at.
func TestExecutorsUpsert(t *testing.T) {
	executors := setup(t)
	ctx := t.Context()
	instanceID := "executor-" + uuid.NewString()

	t.Cleanup(func() {
		require.NoError(t, executors.Delete(context.Background(), &pipelineservice.ExecutorFilter{InstanceID: filter.Equals(instanceID)}))
	})

	first := time.Now().UTC().Truncate(time.Microsecond)
	firstID := uuid.New()
	require.NoError(t, executors.Upsert(ctx, &pipelineservice.Executor{
		ID:          firstID,
		InstanceID:  instanceID,
		Type:        pipelineservice.ExecutorTypeDocker,
		Version:     "v0.5.0",
		Capacity:    5,
		FirstSeenAt: first,
		LastSeenAt:  first,
	}))

	second := first.Add(15 * time.Second)
	require.NoError(t, executors.Upsert(ctx, &pipelineservice.Executor{
		ID:           uuid.New(),
		InstanceID:   instanceID,
		Type:         pipelineservice.ExecutorTypeK8s,
		Version:      "v0.6.0",
		Capacity:     0,
		Capabilities: pipelineservice.ExecutorCapabilities{NativeDestinations: []string{"postgres"}},
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{
			CPURequest: "750m", CPULimit: "750m", CPUExecutorDefault: true,
		},
		FirstSeenAt: second,
		LastSeenAt:  second,
	}))

	rows, err := executors.Find(ctx, &pipelineservice.ExecutorFilter{InstanceID: filter.Equals(instanceID)})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	got := rows[0]
	assert.Equal(t, firstID, got.ID)
	assert.True(t, first.Equal(got.FirstSeenAt), "first_seen_at %s, want %s", got.FirstSeenAt, first)
	assert.True(t, second.Equal(got.LastSeenAt), "last_seen_at %s, want %s", got.LastSeenAt, second)
	assert.Equal(t, pipelineservice.ExecutorTypeK8s, got.Type)
	assert.Equal(t, "v0.6.0", got.Version)
	assert.Equal(t, 0, got.Capacity)
	assert.Equal(t, []string{"postgres"}, got.Capabilities.NativeDestinations)
	assert.Equal(t, pipelineservice.ExecutorResourceDefaults{CPURequest: "750m", CPULimit: "750m", CPUExecutorDefault: true}, got.ResourceDefaults)
}
