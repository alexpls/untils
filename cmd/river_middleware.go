package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/alexpls/untils/internal/wideevents"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type JobLogEvent struct {
	Name     string
	Queue    string
	Attempt  int
	Duration time.Duration
}

func (e *JobLogEvent) Key() string {
	return "job"
}

func (e *JobLogEvent) SlogAttr() slog.Attr {
	return slog.Group(e.Key(),
		slog.String("name", e.Name),
		slog.String("queue", e.Queue),
		slog.Int("attempt", e.Attempt),
		slog.Duration("duration", e.Duration),
	)
}

type wideEventMiddleware struct {
	river.MiddlewareDefaults
	logger *slog.Logger
}

func newWideEventMiddleware(logger *slog.Logger) *wideEventMiddleware {
	return &wideEventMiddleware{logger: logger}
}

func (w *wideEventMiddleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	events := make(wideevents.Events)
	ctx = wideevents.ContextWithEvents(ctx, events)

	ev := wideevents.GetOrCreate(events, func() *JobLogEvent {
		return &JobLogEvent{}
	})
	start := time.Now()

	ev.Name = job.Kind

	err := doInner(ctx)

	ev.Attempt = job.Attempt
	ev.Duration = time.Since(start)
	ev.Queue = job.Queue

	w.logger.LogAttrs(ctx, slog.LevelInfo, "job processed", events.SlogAttrs()...)

	return err
}

var _ rivertype.WorkerMiddleware = &wideEventMiddleware{}
