package bus_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type pipelineStore struct {
	pipelineservice.Storage
	jobs        *jobStorage
	attempts    *attemptStorage
	connections *connectionStorage
	outbox      *recordingOutbox
}

func newPipelineStore(job *pipelineservice.Job, conn *pipelineservice.Connection) *pipelineStore {
	return &pipelineStore{
		jobs:        &jobStorage{job: job},
		attempts:    &attemptStorage{},
		connections: &connectionStorage{conn: conn},
		outbox:      &recordingOutbox{},
	}
}

func (s *pipelineStore) Jobs() pipelineservice.JobsStorage {
	return s.jobs
}

func (s *pipelineStore) JobAttempts() pipelineservice.JobAttemptsStorage {
	return s.attempts
}

func (s *pipelineStore) Connections() pipelineservice.ConnectionsStorage {
	return s.connections
}

func (s *pipelineStore) PipelineEvents() pipelineservice.PipelineEventsOutbox {
	return s.outbox
}

func (s *pipelineStore) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

type jobStorage struct {
	pipelineservice.JobsStorage
	mu  sync.Mutex
	job *pipelineservice.Job
}

func (s *jobStorage) First(_ context.Context, _ *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := s.job.Copy()

	return &cp, nil
}

func (s *jobStorage) Update(_ context.Context, job *pipelineservice.Job) (*pipelineservice.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := job.Copy()
	s.job = &cp

	return job, nil
}

type attemptStorage struct {
	pipelineservice.JobAttemptsStorage
	mu       sync.Mutex
	attempts []*pipelineservice.JobAttempt
}

func (s *attemptStorage) Create(_ context.Context, attempt *pipelineservice.JobAttempt) (*pipelineservice.JobAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts = append(s.attempts, attempt)

	return attempt, nil
}

type connectionStorage struct {
	pipelineservice.ConnectionsStorage
	mu   sync.Mutex
	conn *pipelineservice.Connection
}

func (s *connectionStorage) First(_ context.Context, _ *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := s.conn.Copy()

	return &cp, nil
}

func (s *connectionStorage) Update(_ context.Context, conn *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := conn.Copy()
	s.conn = &cp

	return conn, nil
}

type recordingOutbox struct {
	mu     sync.Mutex
	events []*pipelineservice.PipelineEvent
}

func (o *recordingOutbox) Send(_ context.Context, event *pipelineservice.PipelineEvent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.events = append(o.events, event)

	return nil
}

func (o *recordingOutbox) recorded() []*pipelineservice.PipelineEvent {
	o.mu.Lock()
	defer o.mu.Unlock()

	return append([]*pipelineservice.PipelineEvent(nil), o.events...)
}

type unlimitedRetention struct{}

func (unlimitedRetention) Execute(_ context.Context, workspaceID uuid.UUID) (*pipelineservice.WorkspaceSettings, error) {
	return &pipelineservice.WorkspaceSettings{WorkspaceID: workspaceID, MaxJobsPerWorkspace: 0}, nil
}

func (unlimitedRetention) DeleteOldestTerminalJobs(context.Context, uuid.UUID, int) (int64, error) {
	return 0, nil
}

type notifyStore struct {
	notifyservice.Storage
	webhooks *webhookStorage
	rules    *ruleStorage
}

func newNotifyStore(webhook *notifyservice.Webhook) *notifyStore {
	return &notifyStore{
		webhooks: &webhookStorage{webhook: webhook},
		rules:    &ruleStorage{},
	}
}

func (s *notifyStore) Webhooks() notifyservice.WebhooksStorage {
	return s.webhooks
}

func (s *notifyStore) NotificationRules() notifyservice.NotificationRulesStorage {
	return s.rules
}

type webhookStorage struct {
	notifyservice.WebhooksStorage
	webhook *notifyservice.Webhook
}

func (s *webhookStorage) Find(_ context.Context, _ *notifyservice.WebhookFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*notifyservice.Webhook, error) {
	cp := s.webhook.Copy()

	return []*notifyservice.Webhook{&cp}, nil
}

type ruleStorage struct {
	notifyservice.NotificationRulesStorage
}

func (s *ruleStorage) Find(context.Context, *notifyservice.NotificationRuleFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*notifyservice.NotificationRule, error) {
	return nil, nil
}

type noJobInfo struct{}

func (noJobInfo) LoadJobInfo(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*notifyservice.JobInfo, error) {
	return nil, nil
}

// blockingTransport holds every request until release is closed, then answers
// 200. Each request entering RoundTrip is announced on entered.
type blockingTransport struct {
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
	mu      sync.Mutex
	bodies  [][]byte
}

func newBlockingTransport() *blockingTransport {
	return &blockingTransport{
		entered: make(chan struct{}, 16),
		release: make(chan struct{}),
	}
}

func (b *blockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	b.calls.Add(1)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	b.mu.Lock()
	b.bodies = append(b.bodies, body)
	b.mu.Unlock()

	b.entered <- struct{}{}

	select {
	case <-b.release:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func (b *blockingTransport) requestBodies() [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([][]byte(nil), b.bodies...)
}

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

func testLogger() *logging.Logger {
	return &logging.Logger{Delegate: noopLogDelegate{}}
}
