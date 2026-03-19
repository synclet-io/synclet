package pipelinemetrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector implements prometheus.Collector and provides observation
// methods for the pipeline sync lifecycle. Follows the per-module Collector
// pattern: create with NewMetricsCollector, register via pnpprometheus, and
// call Observe* methods from use cases.
type MetricsCollector struct {
	// Sync lifecycle
	syncsTotal         *prometheus.CounterVec
	syncDuration       *prometheus.HistogramVec
	recordsSyncedTotal *prometheus.CounterVec
	bytesSyncedTotal   *prometheus.CounterVec
	activeSyncs        prometheus.Gauge

	// Health alert metrics
	failedSyncsTotal         *prometheus.CounterVec
	consecutiveFailuresTotal *prometheus.CounterVec
	zeroRecordSyncsTotal     *prometheus.CounterVec

	// Scheduler metrics
	pendingJobs prometheus.Gauge

	// Connector-level metrics
	connectorSyncDuration *prometheus.HistogramVec
}

// NewMetricsCollector creates a MetricsCollector with all pipeline metrics.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		syncsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "synclet_syncs_total",
				Help: "Total number of syncs by status.",
			},
			[]string{"workspace_id", "connection_id", "status"},
		),
		syncDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "synclet_sync_duration_seconds",
				Help:    "Duration of sync operations in seconds.",
				Buckets: prometheus.ExponentialBuckets(1, 2, 12),
			},
			[]string{"workspace_id", "connection_id"},
		),
		recordsSyncedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "synclet_records_synced_total",
				Help: "Total records synced.",
			},
			[]string{"workspace_id", "connection_id", "direction"},
		),
		bytesSyncedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "synclet_bytes_synced_total",
				Help: "Total bytes synced.",
			},
			[]string{"workspace_id", "connection_id"},
		),
		activeSyncs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "synclet_active_syncs",
			Help: "Number of currently running syncs.",
		}),
		failedSyncsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "synclet_failed_syncs_total",
				Help: "Total number of failed syncs.",
			},
			[]string{"workspace_id", "connection_id"},
		),
		consecutiveFailuresTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "synclet_consecutive_failures_total",
				Help: "Counter of consecutive sync failures per connection.",
			},
			[]string{"workspace_id", "connection_id"},
		),
		zeroRecordSyncsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "synclet_zero_record_syncs_total",
				Help: "Total number of syncs that read zero records.",
			},
			[]string{"workspace_id", "connection_id"},
		),
		pendingJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "synclet_pending_jobs",
			Help: "Number of pending jobs in the queue.",
		}),
		connectorSyncDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "synclet_connector_sync_duration_seconds",
				Help:    "Duration of sync operations by connector type.",
				Buckets: prometheus.ExponentialBuckets(1, 2, 12),
			},
			[]string{"connector_type"},
		),
	}
}

// Describe sends all metric descriptors to the channel.
func (m *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	m.syncsTotal.Describe(ch)
	m.syncDuration.Describe(ch)
	m.recordsSyncedTotal.Describe(ch)
	m.bytesSyncedTotal.Describe(ch)
	m.activeSyncs.Describe(ch)
	m.failedSyncsTotal.Describe(ch)
	m.consecutiveFailuresTotal.Describe(ch)
	m.zeroRecordSyncsTotal.Describe(ch)
	m.pendingJobs.Describe(ch)
	m.connectorSyncDuration.Describe(ch)
}

// Collect sends all metric values to the channel.
func (m *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	m.syncsTotal.Collect(ch)
	m.syncDuration.Collect(ch)
	m.recordsSyncedTotal.Collect(ch)
	m.bytesSyncedTotal.Collect(ch)
	m.activeSyncs.Collect(ch)
	m.failedSyncsTotal.Collect(ch)
	m.consecutiveFailuresTotal.Collect(ch)
	m.zeroRecordSyncsTotal.Collect(ch)
	m.pendingJobs.Collect(ch)
	m.connectorSyncDuration.Collect(ch)
}

// ObserveSyncCompleted records a successful sync completion with all stats.
func (m *MetricsCollector) ObserveSyncCompleted(workspaceID, connectionID string, duration time.Duration, recordsRead, bytes int64) {
	m.syncsTotal.WithLabelValues(workspaceID, connectionID, "completed").Inc()
	m.syncDuration.WithLabelValues(workspaceID, connectionID).Observe(duration.Seconds())
	m.recordsSyncedTotal.WithLabelValues(workspaceID, connectionID, "read").Add(float64(recordsRead))
	m.bytesSyncedTotal.WithLabelValues(workspaceID, connectionID).Add(float64(bytes))
}

// ObserveSyncFailed records a sync failure.
func (m *MetricsCollector) ObserveSyncFailed(workspaceID, connectionID string) {
	m.syncsTotal.WithLabelValues(workspaceID, connectionID, "failed").Inc()
	m.failedSyncsTotal.WithLabelValues(workspaceID, connectionID).Inc()
}

// ObserveZeroRecordSync records a sync that read zero records.
func (m *MetricsCollector) ObserveZeroRecordSync(workspaceID, connectionID string) {
	m.zeroRecordSyncsTotal.WithLabelValues(workspaceID, connectionID).Inc()
}

// ObserveActiveSyncStarted increments the active syncs gauge.
func (m *MetricsCollector) ObserveActiveSyncStarted() {
	m.activeSyncs.Inc()
}

// ObserveActiveSyncStopped decrements the active syncs gauge.
func (m *MetricsCollector) ObserveActiveSyncStopped() {
	m.activeSyncs.Dec()
}

// ObserveJobDequeued decrements the pending jobs gauge.
func (m *MetricsCollector) ObserveJobDequeued() {
	m.pendingJobs.Dec()
}
