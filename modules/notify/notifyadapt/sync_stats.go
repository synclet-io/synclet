package notifyadapt

import (
	"encoding/json"
	"time"
)

// syncStatsWire mirrors pipelineservice.SyncStats. The Duration field is a
// time.Duration (nanoseconds) when marshalled by encoding/json.
type syncStatsWire struct {
	RecordsRead int64         `json:"records_read"`
	BytesSynced int64         `json:"bytes_synced"`
	Duration    time.Duration `json:"duration"`
}

type parsedStats struct {
	RecordsRead int64
	BytesSynced int64
	DurationMs  int64
}

func parseSyncStats(raw string) (*parsedStats, error) {
	if raw == "" || raw == "{}" {
		return &parsedStats{}, nil
	}

	var wire syncStatsWire
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return nil, err
	}

	return &parsedStats{
		RecordsRead: wire.RecordsRead,
		BytesSynced: wire.BytesSynced,
		DurationMs:  wire.Duration.Milliseconds(),
	}, nil
}
