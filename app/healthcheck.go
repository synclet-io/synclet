package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/healthcheck/pnphealthcheck"
	"github.com/go-pnp/go-pnp/http/pnphttphealthcheck"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/pkg/ordering"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"github.com/gorilla/mux"
	"go.uber.org/fx"
)

// shutdownTimeout bounds how long every run mode waits for in-flight work on
// stop. Keep it below the chart's terminationGracePeriodSeconds (120s) so the
// process exits before Kubernetes sends SIGKILL.
const shutdownTimeout = 110 * time.Second

const (
	livenessPath  = "/live"
	readinessPath = "/ready"
)

// healthcheckModule serves /live (the process answers HTTP) and /ready (every
// registered health checker passes: database ping, background jobs started,
// event router running) on the internal HTTP server. jobs and consumers add
// the checkers of the background jobs and of the event router.
func healthcheckModule(jobs, consumers bool) fx.Option {
	return fx.Options(
		pnphttphealthcheck.Module(
			pnphttphealthcheck.WithFxPrivate(),
			pnphttphealthcheck.WithEndpoint(readinessPath, http.MethodGet),
		),
		fx.Provide(
			pnphttpserver.MuxHandlerRegistrarProvider(newLivenessRegistrar),
			fx.Private,
		),
		conditionalFxOptions(jobs, jobsHealthModule),
		conditionalFxOptions(consumers, routerHealthModule),
	)
}

func newLivenessRegistrar() pnphttpserver.MuxHandlerRegistrar {
	return pnphttpserver.MuxHandlerRegistrarFunc(func(router *mux.Router) {
		router.Methods(http.MethodGet).Path(livenessPath).HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})
}

func jobsHealthModule() fx.Option {
	return fx.Options(
		fx.Provide(
			newJobsHealth,
			pnpjobber.JobDecoratorProvider(newJobsHealthDecorator),
			pnphealthcheck.HealthCheckerProvider(newJobsHealthChecker),
		),
	)
}

// jobsHealth counts jobber jobs that were registered and those whose Init
// returned, so readiness holds until every background job is running.
type jobsHealth struct {
	registered  atomic.Int32
	initialized atomic.Int32
}

func newJobsHealth() *jobsHealth {
	return &jobsHealth{}
}

func (h *jobsHealth) check(context.Context) error {
	registered, initialized := h.registered.Load(), h.initialized.Load()
	if initialized < registered {
		return fmt.Errorf("%d of %d background jobs started", initialized, registered)
	}

	return nil
}

func newJobsHealthDecorator(h *jobsHealth) ordering.OrderedItem[pnpjobber.JobDecorator] {
	return ordering.Ordered[pnpjobber.JobDecorator](0, func(job jobber.Job) jobber.Job {
		h.registered.Add(1)

		return &healthTrackedJob{Job: job, health: h}
	})
}

func newJobsHealthChecker(h *jobsHealth) pnphealthcheck.HealthChecker {
	return pnphealthcheck.HealthChecker{Name: "jobs", Check: h.check}
}

type healthTrackedJob struct {
	jobber.Job
	health *jobsHealth
}

func (j *healthTrackedJob) Init(ctx context.Context) error {
	if err := j.Job.Init(ctx); err != nil {
		return err
	}

	j.health.initialized.Add(1)

	return nil
}

func routerHealthModule() fx.Option {
	return fx.Provide(pnphealthcheck.HealthCheckerProvider(newRouterHealthChecker))
}

func newRouterHealthChecker(router *message.Router) pnphealthcheck.HealthChecker {
	return pnphealthcheck.HealthChecker{
		Name: "event_router",
		Check: func(context.Context) error {
			if !router.IsRunning() {
				return errors.New("event router is not running")
			}

			return nil
		},
	}
}
