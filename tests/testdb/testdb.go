// Package testdb prepares the Postgres database that database tests share.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/stretchr/testify/require"
)

const lockRetryInterval = 50 * time.Millisecond

// Migrate applies the migrations in fsys, recording them in table. It holds a
// Postgres session lock while it runs, so test packages that run in parallel
// against one database apply migrations one at a time.
func Migrate(t *testing.T, sqlDB *sql.DB, fsys fs.FS, table string) {
	t.Helper()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys,
		goose.WithTableName(table),
		goose.WithSessionLocker(sessionLocker{}),
	)
	require.NoError(t, err)

	_, err = provider.Up(t.Context())
	require.NoError(t, err)
}

// sessionLocker takes goose's advisory lock, retrying every lockRetryInterval
// until it is granted or the context ends.
type sessionLocker struct{}

func (sessionLocker) SessionLock(ctx context.Context, conn *sql.Conn) error {
	ticker := time.NewTicker(lockRetryInterval)
	defer ticker.Stop()

	for {
		var locked bool
		if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lock.DefaultLockID).Scan(&locked); err != nil {
			return fmt.Errorf("pg_try_advisory_lock: %w", err)
		}

		if locked {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (sessionLocker) SessionUnlock(ctx context.Context, conn *sql.Conn) error {
	var unlocked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", lock.DefaultLockID).Scan(&unlocked); err != nil {
		return fmt.Errorf("pg_advisory_unlock: %w", err)
	}

	if !unlocked {
		return fmt.Errorf("advisory lock %d was not held", lock.DefaultLockID)
	}

	return nil
}
