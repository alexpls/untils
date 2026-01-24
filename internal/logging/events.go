package logging

import (
	"context"
	"log/slog"
)

type contextKey int

const (
	_ contextKey = iota
	eventsKey
)

type Events map[string]Event

type Event interface {
	Key() string
	SlogAttr() slog.Attr
}

func (e Events) SlogAttrs() []slog.Attr {
	attrs := make([]slog.Attr, 0, len(e))
	for _, event := range e {
		attrs = append(attrs, event.SlogAttr())
	}
	return attrs
}

// ContextWithEvents returns a new context with wide events enabled.
func ContextWithEvents(ctx context.Context, events Events) context.Context {
	return context.WithValue(ctx, eventsKey, events)
}

// EventsFromContext retrieves the Events map from context.
func EventsFromContext(ctx context.Context) (Events, bool) {
	events, ok := ctx.Value(eventsKey).(Events)
	return events, ok
}

// GetOrCreate retrieves an existing event by key, or creates and stores a new one.
// Returns the event (existing or newly created).
func GetOrCreate[T Event](e Events, create func() T) T {
	var zero T
	key := zero.Key()
	if existing, ok := e[key].(T); ok {
		return existing
	}
	new := create()
	e[new.Key()] = new
	return new
}

// GetOrCreateFromContext retrieves an existing event from context, or creates and stores a new one.
// Always returns a valid event instance (never nil for pointer types).
// The bool indicates whether wide events are enabled - if false, the event won't be
// included in the final log output, but can still be used safely.
func GetOrCreateFromContext[T Event](ctx context.Context, create func() T) (T, bool) {
	e, ok := EventsFromContext(ctx)
	if !ok {
		// Wide events not enabled, but still return a usable instance
		return create(), false
	}
	return GetOrCreate(e, create), true
}
