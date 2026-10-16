package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeJob struct {
	initErr error
}

func (j *fakeJob) Name() string                 { return "fake" }
func (j *fakeJob) Init(context.Context) error   { return j.initErr }
func (j *fakeJob) Handle(context.Context) error { return nil }
func (j *fakeJob) Timer() *time.Timer           { return time.NewTimer(time.Hour) }
func (j *fakeJob) ResetTimer(timer *time.Timer) { timer.Reset(time.Hour) }

func TestJobsHealthNotReadyUntilEveryJobInitialized(t *testing.T) {
	health := newJobsHealth()
	decorate := newJobsHealthDecorator(health).Value

	first := decorate(&fakeJob{})
	second := decorate(&fakeJob{initErr: errors.New("boom")})

	if err := health.check(context.Background()); err == nil {
		t.Fatal("ready before any job initialized")
	}

	if err := first.Init(context.Background()); err != nil {
		t.Fatalf("init first: %v", err)
	}

	if err := second.Init(context.Background()); err == nil {
		t.Fatal("expected init error from second job")
	}

	if err := health.check(context.Background()); err == nil {
		t.Fatal("ready while a job failed to initialize")
	}
}

func TestJobsHealthReadyWithoutJobs(t *testing.T) {
	if err := newJobsHealth().check(context.Background()); err != nil {
		t.Fatalf("no jobs registered: %v", err)
	}
}
