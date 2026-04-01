package pipelineservice

import (
	"time"

	"github.com/robfig/cron/v3"
)

// CronParser is the shared cron expression parser used throughout the pipeline module.
// Flags: Minute | Hour | Dom | Month | Dow (standard 5-field cron).
var CronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ComputeNextScheduledAt computes the next scheduled time for a cron expression.
// Returns nil if schedule is nil, empty, or unparseable.
func ComputeNextScheduledAt(schedule *string, from time.Time) *time.Time {
	if schedule == nil || *schedule == "" {
		return nil
	}

	sched, err := CronParser.Parse(*schedule)
	if err != nil {
		return nil
	}

	next := sched.Next(from)

	return &next
}

// RecomputeNextScheduledAt updates conn.NextScheduledAt based on connection status and schedule.
// If the connection is active, computes the next cron tick; otherwise sets nil.
// Call this whenever status or schedule changes.
func RecomputeNextScheduledAt(conn *Connection, now time.Time) {
	if conn.Status == ConnectionStatusActive {
		conn.NextScheduledAt = ComputeNextScheduledAt(conn.Schedule, now)
	} else {
		conn.NextScheduledAt = nil
	}
}
