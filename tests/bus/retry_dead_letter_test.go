package bus_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/eventbus"
)

const (
	sourceTopic = "events.test"
	handlerName = "test_handler"
	maxRetries  = 3
)

var errHandler = errors.New("handler failed")

type busFixture struct {
	pubSub      *gochannel.GoChannel
	router      *message.Router
	deadLetters <-chan *message.Message
	calls       atomic.Int32
}

func newBusFixture(t *testing.T, handle func(attempt int32) error) *busFixture {
	t.Helper()

	fixture := &busFixture{
		pubSub: gochannel.NewGoChannel(gochannel.Config{}, watermill.NopLogger{}),
	}

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	require.NoError(t, err)

	busMiddleware, err := eventbus.Middleware(eventbus.RetryConfig{
		MaxRetries:      maxRetries,
		InitialInterval: time.Millisecond,
		MaxInterval:     5 * time.Millisecond,
	}, fixture.pubSub, watermill.NopLogger{})
	require.NoError(t, err)

	router.AddMiddleware(busMiddleware)
	router.AddConsumerHandler(handlerName, sourceTopic, fixture.pubSub, func(*message.Message) error {
		return handle(fixture.calls.Add(1))
	})

	ctx, cancel := context.WithCancel(context.Background())

	deadLetters, err := fixture.pubSub.Subscribe(ctx, eventbus.DeadLetterTopic)
	require.NoError(t, err)

	fixture.router = router
	fixture.deadLetters = deadLetters

	go func() {
		_ = router.Run(ctx)
	}()

	<-router.Running()

	t.Cleanup(func() {
		cancel()
		assert.NoError(t, router.Close())
		assert.NoError(t, fixture.pubSub.Close())
	})

	return fixture
}

func (f *busFixture) publish(t *testing.T) *message.Message {
	t.Helper()

	msg := message.NewMessage(uuid.NewString(), []byte(`{"job_id":"42"}`))
	msg.Metadata.Set("event_type", "job.updated")

	require.NoError(t, f.pubSub.Publish(sourceTopic, msg))

	return msg
}

func (f *busFixture) waitForDeadLetter(t *testing.T) *message.Message {
	t.Helper()

	select {
	case dead := <-f.deadLetters:
		dead.Ack()

		return dead
	case <-time.After(5 * time.Second):
		t.Fatal("no message arrived on the dead-letter topic")

		return nil
	}
}

func (f *busFixture) assertNoDeadLetter(t *testing.T) {
	t.Helper()

	select {
	case dead := <-f.deadLetters:
		dead.Ack()
		t.Fatalf("unexpected dead-letter message %s", dead.UUID)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestMiddleware_RetriesThenDeadLetters(t *testing.T) {
	fixture := newBusFixture(t, func(int32) error {
		return errHandler
	})

	original := fixture.publish(t)
	dead := fixture.waitForDeadLetter(t)

	assert.Equal(t, int32(maxRetries+1), fixture.calls.Load())
	assert.Equal(t, original.UUID, dead.UUID)
	assert.Equal(t, original.Payload, dead.Payload)
	assert.Equal(t, "job.updated", dead.Metadata.Get("event_type"))
	assert.Equal(t, errHandler.Error(), dead.Metadata.Get(middleware.ReasonForPoisonedKey))
	assert.Equal(t, sourceTopic, dead.Metadata.Get(middleware.PoisonedTopicKey))
	assert.Equal(t, handlerName, dead.Metadata.Get(middleware.PoisonedHandlerKey))

	fixture.assertNoDeadLetter(t)
}

func TestMiddleware_RecoversWithinRetryBudget(t *testing.T) {
	succeeded := make(chan struct{})

	fixture := newBusFixture(t, func(attempt int32) error {
		if attempt <= 2 {
			return errHandler
		}

		close(succeeded)

		return nil
	})

	fixture.publish(t)

	select {
	case <-succeeded:
	case <-time.After(5 * time.Second):
		t.Fatal("handler never succeeded")
	}

	fixture.assertNoDeadLetter(t)
	assert.Equal(t, int32(3), fixture.calls.Load())
}
