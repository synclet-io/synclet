// Package eventbus holds the delivery policy shared by every event consumer:
// a failing handler is retried with exponential backoff, and once the retries
// are exhausted the message is published to DeadLetterTopic and acked.
package eventbus

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// DeadLetterTopic receives messages whose handler failed on every retry.
// The dead-lettered message keeps its UUID, payload and metadata, plus the
// middleware.ReasonForPoisonedKey, PoisonedTopicKey, PoisonedHandlerKey and
// PoisonedSubscriberKey metadata describing where and why it failed.
const DeadLetterTopic = "events.dead_letter"

// BackoffMultiplier scales the wait between consecutive retries.
const BackoffMultiplier = 2.0

// RetryConfig bounds the retries applied to a failing handler before the
// message is dead-lettered.
type RetryConfig struct {
	// MaxRetries is the number of attempts after the first one.
	MaxRetries int
	// InitialInterval is the wait before the first retry.
	InitialInterval time.Duration
	// MaxInterval caps the wait between retries.
	MaxInterval time.Duration
}

// Middleware returns the handler middleware applying the delivery policy:
// retries with exponential backoff wrapped by dead-lettering, so the
// dead-letter publish happens only after the last retry has failed. The
// handler error is swallowed once the message is dead-lettered, which acks
// it; if the dead-letter publish itself fails the message is nacked and
// redelivered.
func Middleware(cfg RetryConfig, deadLetterPublisher message.Publisher, logger watermill.LoggerAdapter) (message.HandlerMiddleware, error) {
	deadLetter, err := middleware.PoisonQueue(deadLetterPublisher, DeadLetterTopic)
	if err != nil {
		return nil, fmt.Errorf("create dead-letter middleware: %w", err)
	}

	retry := middleware.Retry{
		MaxRetries:      cfg.MaxRetries,
		InitialInterval: cfg.InitialInterval,
		MaxInterval:     cfg.MaxInterval,
		Multiplier:      BackoffMultiplier,
		Logger:          logger,
	}

	return func(handler message.HandlerFunc) message.HandlerFunc {
		return deadLetter(retry.Middleware(handler))
	}, nil
}
