package app

import (
	"log/slog"

	"go.uber.org/fx/fxevent"
)

type FxLogger struct {
	logger *slog.Logger
}

func (f FxLogger) LogEvent(event fxevent.Event) {
	if event, ok := event.(*fxevent.Started); ok {
		if event.Err != nil {
			f.logger.Error("app started with error", slog.Any("error", event.Err))
		}
	}
}
