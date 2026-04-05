package pipelineservice

import "time"

// Config holds pipeline service configuration injected from the app layer.
type Config struct {
	RuntimeDefaults             RuntimeDefaults
	IdleTimeout                 time.Duration
	MaxConcurrentJobs           int
	ConnectorTaskRetention      time.Duration
	ConnectorTaskPendingTimeout time.Duration
	ConnectorTaskRunningTimeout time.Duration
	DefaultMaxAttempts          int
	ConnectionCheckTimeout      time.Duration
	ConnectorTaskPollInterval   time.Duration
	EventEmitTimeout            time.Duration
	StatsRollupHourlyLookback   time.Duration
	StatsRollupDailyLookback    time.Duration
	RegistryFetchTimeout        time.Duration
	HeartbeatTimeout            time.Duration
}
