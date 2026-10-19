package app

import (
	"context"
	"os"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/pkg/migrations"
)

// TestWorkspaceBootstrap starts the workspace module with the run options of
// `synclet server` on an empty database and checks which default workspace
// registration finds. It needs TEST_DATABASE_DSN with a role that may create
// databases.
func TestWorkspaceBootstrap(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping workspace bootstrap test")
	}

	t.Run("single mode creates the default workspace that registrations join", func(t *testing.T) {
		t.Setenv("WORKSPACE_MODE", "single")

		db := startWorkspaceModule(t, dsn)
		storage := newWorkspaceStorage(db, nil)

		workspace, err := storage.Workspaces().First(t.Context(), &workspaceservice.WorkspaceFilter{
			Slug: filter.Equals("default"),
		})
		require.NoError(t, err)

		autoAssign := workspaceservice.NewAutoAssignMember(storage)
		first, second := uuid.New(), uuid.New()
		require.NoError(t, autoAssign.Execute(t.Context(), first))
		require.NoError(t, autoAssign.Execute(t.Context(), second))

		members, err := storage.WorkspaceMembers().Find(t.Context(), &workspaceservice.WorkspaceMemberFilter{
			WorkspaceID: filter.Equals(workspace.ID),
		})
		require.NoError(t, err)

		roles := map[uuid.UUID]workspaceservice.MemberRole{}
		for _, member := range members {
			roles[member.UserID] = member.Role
		}

		assert.Equal(t, map[uuid.UUID]workspaceservice.MemberRole{
			first:  workspaceservice.MemberRoleAdmin,
			second: workspaceservice.MemberRoleViewer,
		}, roles)
	})

	t.Run("multi mode creates no workspace", func(t *testing.T) {
		t.Setenv("WORKSPACE_MODE", "multi")

		db := startWorkspaceModule(t, dsn)

		count, err := newWorkspaceStorage(db, nil).Workspaces().Count(t.Context(), &workspaceservice.WorkspaceFilter{})
		require.NoError(t, err)
		assert.Zero(t, count)
	})
}

// startWorkspaceModule applies every module's migrations to a scratch database and starts the workspace
// module on it with the options of `synclet server` without --standalone.
func startWorkspaceModule(t *testing.T, dsn string) *gorm.DB {
	t.Helper()

	sqlDB := stdlib.OpenDB(*createScratchDatabase(t, dsn))
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	for _, source := range migrations.Sources() {
		provider, err := newMigrationProvider(sqlDB, source)
		require.NoError(t, err)

		_, err = provider.Up(t.Context())
		require.NoError(t, err)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	options := optionutil.ApplyOptions(serverDefaultOptions(), ServerCommandOptions(false)...)
	app := fx.New(
		fx.NopLogger,
		fx.Supply(db, &logging.Logger{Delegate: nopLogDelegate{}}),
		workspaceModule(options),
	)
	require.NoError(t, app.Err())

	require.NoError(t, app.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, app.Stop(context.WithoutCancel(t.Context()))) })

	return db
}

type nopLogDelegate struct{}

func (nopLogDelegate) Info(context.Context, string, ...any)         {}
func (nopLogDelegate) Warn(context.Context, string, ...any)         {}
func (nopLogDelegate) Debug(context.Context, string, ...any)        {}
func (nopLogDelegate) Error(context.Context, string, ...any)        {}
func (d nopLogDelegate) WithFields(map[string]any) logging.Delegate { return d }
func (d nopLogDelegate) WithField(string, any) logging.Delegate     { return d }
func (d nopLogDelegate) WithError(error) logging.Delegate           { return d }
func (d nopLogDelegate) Named(string) logging.Delegate              { return d }
func (d nopLogDelegate) SkipCallers(int) logging.Delegate           { return d }
