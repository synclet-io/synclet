package pipelinestats_test

import (
	"context"
	"errors"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

var errStorage = errors.New("storage failure")

type window struct {
	from, to time.Time
}

func (w window) contains(t time.Time) bool {
	return !t.Before(w.from) && t.Before(w.to)
}

// rollupBucket is a rollup row positioned relative to the upper bound of the
// first rollup query, so tests can place rows exactly on window edges even
// though the use cases read the clock themselves.
type rollupBucket struct {
	connectionID    uuid.UUID
	offset          time.Duration
	syncsSucceeded  int64
	syncsFailed     int64
	recordsRead     int64
	totalDurationMs int64
}

type upsertCall struct {
	bucketSize pipelineservice.BucketSize
	since      time.Time
	truncUnit  string
}

type timelineCall struct {
	workspaceID  uuid.UUID
	bucketSize   pipelineservice.BucketSize
	window       window
	connectionID *uuid.UUID
}

type fakeStatsStorage struct {
	pipelineservice.StatsStorage

	errs map[string]error

	buckets []rollupBucket
	anchor  *time.Time

	lastSyncAt     *time.Time
	lastJobInfo    *pipelineservice.LastJobInfo
	durationRows   []pipelineservice.DurationChartRow
	recordRows     []pipelineservice.RecordsChartRow
	failedJobs     []pipelineservice.FailedJobRow
	timelineRows   []pipelineservice.TimelineRow
	topConnections []pipelineservice.TopConnectionRow
	lastJobInfos   map[uuid.UUID]*pipelineservice.LastJobInfo
	lastCompleted  map[uuid.UUID]*time.Time
	sparklines     map[uuid.UUID][]int64

	upserts          []upsertCall
	rollupWindows    []window
	durationWindow   window
	durationLimit    int
	recordsWindow    window
	failedJobsWindow window
	timelineCalls    []timelineCall
	topWindow        window
	topLimit         int
	sparklineLimit   int
	lastJobInfoBatch []uuid.UUID
	queriedLastJob   bool
	bucketSizes      map[string][]pipelineservice.BucketSize
}

func (s *fakeStatsStorage) recordBucketSize(method string, bucketSize pipelineservice.BucketSize) {
	if s.bucketSizes == nil {
		s.bucketSizes = map[string][]pipelineservice.BucketSize{}
	}

	s.bucketSizes[method] = append(s.bucketSizes[method], bucketSize)
}

func (s *fakeStatsStorage) err(method string) error {
	return s.errs[method]
}

func (s *fakeStatsStorage) aggregate(connectionID *uuid.UUID, from, to time.Time) pipelineservice.ConnectionRollup {
	if s.anchor == nil {
		s.anchor = &to
	}

	w := window{from: from, to: to}
	s.rollupWindows = append(s.rollupWindows, w)

	var out pipelineservice.ConnectionRollup

	for _, b := range s.buckets {
		if connectionID != nil && b.connectionID != *connectionID {
			continue
		}

		if !w.contains(s.anchor.Add(b.offset)) {
			continue
		}

		out.SyncsSucceeded += b.syncsSucceeded
		out.SyncsFailed += b.syncsFailed
		out.SyncsTotal += b.syncsSucceeded + b.syncsFailed
		out.RecordsRead += b.recordsRead
		out.TotalDurationMs += b.totalDurationMs
	}

	return out
}

func (s *fakeStatsStorage) UpsertRollups(_ context.Context, bucketSize pipelineservice.BucketSize, since time.Time, truncUnit string) error {
	s.upserts = append(s.upserts, upsertCall{bucketSize: bucketSize, since: since, truncUnit: truncUnit})

	return s.err("UpsertRollups:" + truncUnit)
}

func (s *fakeStatsStorage) QueryConnectionRollup(_ context.Context, connectionID uuid.UUID, bucketSize pipelineservice.BucketSize, from, to time.Time) (pipelineservice.ConnectionRollup, error) {
	s.recordBucketSize("QueryConnectionRollup", bucketSize)

	if err := s.err("QueryConnectionRollup"); err != nil {
		return pipelineservice.ConnectionRollup{}, err
	}

	rollup := s.aggregate(&connectionID, from, to)
	if len(s.rollupWindows) == 2 {
		if err := s.err("QueryConnectionRollup:previous"); err != nil {
			return pipelineservice.ConnectionRollup{}, err
		}
	}

	return rollup, nil
}

func (s *fakeStatsStorage) QueryWorkspaceRollup(_ context.Context, _ uuid.UUID, bucketSize pipelineservice.BucketSize, from, to time.Time) (pipelineservice.RollupTotals, error) {
	s.recordBucketSize("QueryWorkspaceRollup", bucketSize)

	if err := s.err("QueryWorkspaceRollup"); err != nil {
		return pipelineservice.RollupTotals{}, err
	}

	rollup := s.aggregate(nil, from, to)
	if len(s.rollupWindows) == 2 {
		if err := s.err("QueryWorkspaceRollup:previous"); err != nil {
			return pipelineservice.RollupTotals{}, err
		}
	}

	return pipelineservice.RollupTotals{
		SyncsTotal:     rollup.SyncsTotal,
		SyncsSucceeded: rollup.SyncsSucceeded,
		SyncsFailed:    rollup.SyncsFailed,
		RecordsRead:    rollup.RecordsRead,
	}, nil
}

func (s *fakeStatsStorage) QueryLastSyncAt(context.Context, uuid.UUID) (*time.Time, error) {
	return s.lastSyncAt, s.err("QueryLastSyncAt")
}

func (s *fakeStatsStorage) QueryLastJobInfo(context.Context, uuid.UUID) (*pipelineservice.LastJobInfo, error) {
	s.queriedLastJob = true

	return s.lastJobInfo, s.err("QueryLastJobInfo")
}

func (s *fakeStatsStorage) QueryDurationChart(_ context.Context, _ uuid.UUID, from, to time.Time, limit int) ([]pipelineservice.DurationChartRow, error) {
	s.durationWindow = window{from: from, to: to}
	s.durationLimit = limit

	return s.durationRows, s.err("QueryDurationChart")
}

func (s *fakeStatsStorage) QueryRecordsChart(_ context.Context, _ uuid.UUID, bucketSize pipelineservice.BucketSize, from, to time.Time) ([]pipelineservice.RecordsChartRow, error) {
	s.recordBucketSize("QueryRecordsChart", bucketSize)

	s.recordsWindow = window{from: from, to: to}

	return s.recordRows, s.err("QueryRecordsChart")
}

func (s *fakeStatsStorage) QueryConnectionFailedJobs(_ context.Context, _ uuid.UUID, from, to time.Time) ([]pipelineservice.FailedJobRow, error) {
	s.failedJobsWindow = window{from: from, to: to}

	return s.failedJobs, s.err("QueryConnectionFailedJobs")
}

func (s *fakeStatsStorage) QueryWorkspaceFailedJobs(_ context.Context, _ uuid.UUID, from, to time.Time) ([]pipelineservice.FailedJobRow, error) {
	s.failedJobsWindow = window{from: from, to: to}

	return s.failedJobs, s.err("QueryWorkspaceFailedJobs")
}

func (s *fakeStatsStorage) QuerySyncTimeline(_ context.Context, workspaceID uuid.UUID, bucketSize pipelineservice.BucketSize, from, to time.Time, connectionID *uuid.UUID) ([]pipelineservice.TimelineRow, error) {
	s.timelineCalls = append(s.timelineCalls, timelineCall{
		workspaceID:  workspaceID,
		bucketSize:   bucketSize,
		window:       window{from: from, to: to},
		connectionID: connectionID,
	})

	return s.timelineRows, s.err("QuerySyncTimeline")
}

func (s *fakeStatsStorage) QueryTopConnections(_ context.Context, _ uuid.UUID, bucketSize pipelineservice.BucketSize, from, to time.Time, limit int) ([]pipelineservice.TopConnectionRow, error) {
	s.recordBucketSize("QueryTopConnections", bucketSize)

	s.topWindow = window{from: from, to: to}
	s.topLimit = limit

	return s.topConnections, s.err("QueryTopConnections")
}

func (s *fakeStatsStorage) QueryLastJobInfoBatch(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]*pipelineservice.LastJobInfo, error) {
	s.lastJobInfoBatch = ids
	if err := s.err("QueryLastJobInfoBatch"); err != nil {
		return nil, err
	}

	return s.lastJobInfos, nil
}

func (s *fakeStatsStorage) QueryConnectionLastCompletedAtBatch(context.Context, []uuid.UUID) (map[uuid.UUID]*time.Time, error) {
	if err := s.err("QueryConnectionLastCompletedAtBatch"); err != nil {
		return nil, err
	}

	return s.lastCompleted, nil
}

func (s *fakeStatsStorage) QueryConnectionSparklineBatch(_ context.Context, _ []uuid.UUID, bucketSize pipelineservice.BucketSize, _, _ time.Time, limit int) (map[uuid.UUID][]int64, error) {
	s.recordBucketSize("QueryConnectionSparklineBatch", bucketSize)

	s.sparklineLimit = limit
	if err := s.err("QueryConnectionSparklineBatch"); err != nil {
		return nil, err
	}

	return s.sparklines, nil
}

type fakeConnectionsStorage struct {
	pipelineservice.ConnectionsStorage

	records  []*pipelineservice.Connection
	firstNil bool
	err      error
}

func matchesConnection(conn *pipelineservice.Connection, f *pipelineservice.ConnectionFilter) bool {
	if id, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.ID != id.Value {
		return false
	}

	if wid, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.WorkspaceID != wid.Value {
		return false
	}

	return true
}

func (s *fakeConnectionsStorage) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	if s.err != nil {
		return nil, s.err
	}

	if s.firstNil {
		return nil, nil //nolint:nilnil // exercises the nil-connection guard
	}

	for _, conn := range s.records {
		if matchesConnection(conn, f) {
			cp := *conn

			return &cp, nil
		}
	}

	return nil, pipelineservice.ErrConnectionNotFound
}

func (s *fakeConnectionsStorage) Find(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	if s.err != nil {
		return nil, s.err
	}

	var out []*pipelineservice.Connection

	for _, conn := range s.records {
		if matchesConnection(conn, f) {
			cp := *conn
			out = append(out, &cp)
		}
	}

	return out, nil
}

type fakeStorage struct {
	pipelineservice.Storage

	connections *fakeConnectionsStorage
}

func (s *fakeStorage) Connections() pipelineservice.ConnectionsStorage {
	return s.connections
}

func newFakeStorage(conns ...*pipelineservice.Connection) *fakeStorage {
	return &fakeStorage{connections: &fakeConnectionsStorage{records: conns}}
}

type noopDelegate struct{}

func (noopDelegate) Info(context.Context, string, ...interface{})         {}
func (noopDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopDelegate) Named(string) logging.Delegate                      { return d }
func (d noopDelegate) SkipCallers(int) logging.Delegate                   { return d }

func newLogger() *logging.Logger {
	return &logging.Logger{Delegate: noopDelegate{}}
}

func ptr[T any](v T) *T {
	return &v
}

// minuteAligned returns a sync time on a minute boundary so an every-minute
// schedule yields a full one-minute expected interval.
func minuteAligned() time.Time {
	return time.Now().Truncate(time.Minute)
}

const everyMinute = "* * * * *"
