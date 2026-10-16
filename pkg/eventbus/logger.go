package eventbus

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/go-pnp/go-pnp/logging"
)

type loggerAdapter struct {
	logger *logging.Logger
}

// NewLoggerAdapter exposes an application logger to Watermill.
func NewLoggerAdapter(logger *logging.Logger) watermill.LoggerAdapter {
	return loggerAdapter{logger: logger}
}

func (l loggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	l.logger.WithError(err).WithFields(fields).Error(context.Background(), msg)
}

func (l loggerAdapter) Info(msg string, fields watermill.LogFields) {
	l.logger.WithFields(fields).Info(context.Background(), msg)
}

func (l loggerAdapter) Debug(msg string, fields watermill.LogFields) {
	l.logger.WithFields(fields).Debug(context.Background(), msg)
}

func (l loggerAdapter) Trace(msg string, fields watermill.LogFields) {
	l.logger.WithFields(fields).Debug(context.Background(), msg)
}

func (l loggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return loggerAdapter{logger: l.logger.WithFields(fields)}
}
