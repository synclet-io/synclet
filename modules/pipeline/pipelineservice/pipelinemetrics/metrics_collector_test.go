package pipelinemetrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
)

func TestInitLogArchiveFailures_ExportsBackendAtZero(t *testing.T) {
	metrics := pipelinemetrics.NewMetricsCollector()
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(metrics)

	metrics.InitLogArchiveFailures("s3")

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP synclet_log_archive_failures_total Total number of job attempts whose logs failed to archive, by archive backend.
# TYPE synclet_log_archive_failures_total counter
synclet_log_archive_failures_total{backend="s3"} 0
`), "synclet_log_archive_failures_total"))
}
