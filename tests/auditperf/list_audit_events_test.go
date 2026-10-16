package auditperf_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/synclet-io/synclet/modules/audit/auditdbstate"
	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/audit/auditstorage"
)

const (
	targetEvents = 1000
	noiseEvents  = 200
	actorCount   = 5
	historyDays  = 90
	runs         = 40
	pageSize     = 50
	p95Budget    = 500 * time.Millisecond
)

var (
	allActions = []auditservice.Action{
		auditservice.ActionCreate, auditservice.ActionUpdate, auditservice.ActionDelete,
		auditservice.ActionEnable, auditservice.ActionDisable, auditservice.ActionTest,
		auditservice.ActionSync, auditservice.ActionLogin, auditservice.ActionLogout,
	}
	allResourceTypes = []auditservice.ResourceType{
		auditservice.ResourceTypeSource, auditservice.ResourceTypeDestination,
		auditservice.ResourceTypeConnection, auditservice.ResourceTypeConnector,
		auditservice.ResourceTypeRepository, auditservice.ResourceTypeNotificationChannel,
		auditservice.ResourceTypeNotificationRule, auditservice.ResourceTypeWebhook,
		auditservice.ResourceTypeWorkspace, auditservice.ResourceTypeWorkspaceMember,
		auditservice.ResourceTypeWorkspaceInvite, auditservice.ResourceTypeAPIKey,
		auditservice.ResourceTypeUser,
	}
)

func TestListAuditEventsP95(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping audit list performance test")
	}

	ctx := context.Background()
	db := openDB(t, dsn)
	truncateAuditEvents(t, db)
	t.Cleanup(func() { truncateAuditEvents(t, db) })

	storage := auditstorage.NewStorages(db, nil, nil)
	workspaceID := uuid.New()

	actors := make([]uuid.UUID, actorCount)
	for i := range actors {
		actors[i] = uuid.New()
	}

	now := time.Now().UTC()
	seedEvents(ctx, t, storage, workspaceID, actors, now, targetEvents)
	seedEvents(ctx, t, storage, uuid.New(), []uuid.UUID{uuid.New()}, now, noiseEvents)

	since := now.AddDate(0, 0, -30)
	until := now.AddDate(0, 0, -45)
	resourceType := auditservice.ResourceTypeConnection
	action := auditservice.ActionUpdate

	cases := []auditservice.ListAuditEventsParams{
		{},
		{ActorID: &actors[0]},
		{ResourceType: &resourceType},
		{Action: &action},
		{Since: &since},
		{Until: &until},
		{ActorID: &actors[1], ResourceType: &resourceType, Action: &action, Since: &since},
		{ActorID: &actors[2], Action: &action, Offset: pageSize},
	}

	list := auditservice.NewListAuditEvents(storage)
	durations := make([]time.Duration, 0, runs)

	for i := range runs {
		params := cases[i%len(cases)]
		params.WorkspaceID = workspaceID
		params.Limit = pageSize

		start := time.Now()
		events, err := list.Execute(ctx, params)

		durations = append(durations, time.Since(start))

		require.NoError(t, err)
		require.LessOrEqual(t, len(events), pageSize)

		if params.ActorID == nil && params.ResourceType == nil && params.Action == nil && params.Since == nil && params.Until == nil {
			require.Len(t, events, pageSize)
		}

		for _, event := range events {
			assertMatches(t, params, event)
		}
	}

	slices.Sort(durations)
	p50 := percentile(durations, 50)
	p95 := percentile(durations, 95)
	t.Logf("ListAuditEvents over %d runs: p50=%s p95=%s max=%s", runs, p50, p95, durations[len(durations)-1])
	require.Less(t, p95, p95Budget, "p95 latency exceeds budget")
}

func openDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	goose.SetBaseFS(auditdbstate.MigrationsFS)
	goose.SetTableName("audit_goose_db_version")
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(sqlDB, "."))

	return db
}

func truncateAuditEvents(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec("TRUNCATE audit.audit_events").Error)
}

func seedEvents(
	ctx context.Context,
	t *testing.T,
	storage *auditstorage.Storages,
	workspaceID uuid.UUID,
	actors []uuid.UUID,
	now time.Time,
	count int,
) {
	t.Helper()

	window := time.Duration(historyDays) * 24 * time.Hour

	events := make([]*auditservice.AuditEvent, count)
	for i := range events {
		events[i] = &auditservice.AuditEvent{
			ID:            uuid.New(),
			WorkspaceID:   workspaceID,
			ActorType:     auditservice.ActorTypeUser,
			ActorID:       actors[i%len(actors)],
			ActorLabel:    fmt.Sprintf("actor-%d", i%len(actors)),
			Action:        allActions[i%len(allActions)],
			ResourceType:  allResourceTypes[i%len(allResourceTypes)],
			ResourceID:    uuid.New(),
			ResourceLabel: fmt.Sprintf("resource-%d", i),
			DiffJSON:      `[{"field":"name","old":"a","new":"b"}]`,
			CreatedAt:     now.Add(-window * time.Duration(i) / time.Duration(count)),
		}
	}

	_, err := storage.AuditEvents().BatchCreate(ctx, events)
	require.NoError(t, err)
}

func assertMatches(t *testing.T, params auditservice.ListAuditEventsParams, event *auditservice.AuditEvent) {
	t.Helper()

	require.Equal(t, params.WorkspaceID, event.WorkspaceID)

	if params.ActorID != nil {
		require.Equal(t, *params.ActorID, event.ActorID)
	}

	if params.ResourceType != nil {
		require.Equal(t, *params.ResourceType, event.ResourceType)
	}

	if params.Action != nil {
		require.Equal(t, *params.Action, event.Action)
	}

	if params.Since != nil {
		require.False(t, event.CreatedAt.Before(*params.Since))
	}

	if params.Until != nil && params.Since == nil {
		require.True(t, event.CreatedAt.Before(*params.Until))
	}
}

func percentile(sorted []time.Duration, pct int) time.Duration {
	idx := (len(sorted)*pct+99)/100 - 1

	return sorted[max(idx, 0)]
}
